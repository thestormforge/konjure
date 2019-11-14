/*
Copyright 2019 GramLabs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kustomize

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/types"
	"sigs.k8s.io/kustomize/v3/plugin/builtin"
)

// PluginRunner is used to create Cobra commands that run Kustomize plugins
type PluginRunner struct {
	plugin interface{}
	cmd    *cobra.Command

	root      string
	config    func(*cobra.Command, []string) ([]byte, error)
	generate  func() (resmap.ResMap, error)
	transform func(resMap resmap.ResMap) error

	ldr ifc.Loader
	rf  *resmap.Factory
}

// RunnerOption is an option that can be applied when creating a plugin runner
type RunnerOption func(*PluginRunner)

// NewPluginRunner creates a new runner for the supplied plugin and options
func NewPluginRunner(plugin interface{}, opts ...RunnerOption) *cobra.Command {
	k := &PluginRunner{
		plugin:    plugin,
		config:    func(*cobra.Command, []string) ([]byte, error) { return nil, nil },
		generate:  func() (resmap.ResMap, error) { return resmap.New(), nil },
		transform: func(resmap.ResMap) error { return nil },
	}

	// Setup the command run stages
	k.cmd = &cobra.Command{
		PreRunE:  k.preRun,
		RunE:     k.run,
		PostRunE: k.postRun,
	}

	// Establish generate and transform functions
	if p, ok := plugin.(resmap.Generator); ok {
		k.generate = p.Generate
	}
	if p, ok := plugin.(resmap.Transformer); ok {
		k.generate = k.newResMapFromStdin
		k.transform = p.Transform
	}

	// Apply the runner options
	for _, opt := range opts {
		opt(k)
	}

	return k.cmd
}

// preRun will create the plugin helpers and invoke the configure method of the plugin
func (k *PluginRunner) preRun(cmd *cobra.Command, args []string) error {
	ldr, err := NewKonjureLoader(context.Background(), k.root)
	if err != nil {
		return err
	}
	k.ldr = ldr

	uf := kunstruct.NewKunstructuredFactoryImpl()
	pf := transformer.NewFactoryImpl()
	k.rf = resmap.NewFactory(resource.NewFactory(uf), pf)

	config, err := k.config(cmd, args)
	if err != nil {
		return err
	}

	c, ok := k.plugin.(resmap.Configurable)
	if !ok {
		return nil // Ignore non-configurable plugins
	}

	return c.Config(k.ldr, k.rf, config)
}

// run will actually run everything
func (k *PluginRunner) run(cmd *cobra.Command, args []string) error {
	m, err := k.generate()
	if err != nil {
		return err
	}

	if err := k.transform(m); err != nil {
		return err
	}

	if err := persistResourceOptions(m); err != nil {
		return err
	}

	b, err := m.AsYaml()
	if err != nil {
		return err
	}

	_, err = cmd.OutOrStdout().Write(b)
	return err
}

// postRun will perform necessary clean up
func (k *PluginRunner) postRun(cmd *cobra.Command, args []string) error {
	return k.ldr.Cleanup()
}

// newResMapFromStdin reads stdin and parses it as a resource map
func (k *PluginRunner) newResMapFromStdin() (resmap.ResMap, error) {
	b, err := ioutil.ReadAll(k.cmd.InOrStdin())
	if err != nil {
		return nil, err
	}
	return k.rf.NewResMapFromBytes(b)
}

// addTransformerPlugin will mutate the transform function to also run the supplied plugin
func (k *PluginRunner) addTransformerPlugin(t resmap.TransformerPlugin, config []byte) {
	k.transform = combineTransformFunc(k.transform, func(m resmap.ResMap) error {
		if err := t.Config(k.ldr, k.rf, config); err != nil {
			return err
		}
		return t.Transform(m)
	})
}

// WithConfigType will annotate the Cobra command with the GVK of the configuration schema; it will also setup the
// positional arguments and pre-run of the command to read a configuration file of the specified kind.
func WithConfigType(group, version, kind string) RunnerOption {
	return func(k *PluginRunner) {
		// Record the GVK information on the command
		k.cmd.Use = kind + " FILE"
		k.cmd.Short = fmt.Sprintf("Executable plugin for the %s kind", kind)
		k.cmd.Hidden = true
		k.cmd.Annotations = map[string]string{
			"group":   group,
			"version": version,
			"kind":    kind,
		}

		// TODO We should take an example object and serialize it as the example text

		// Require an argument for the configuration filename
		k.cmd.Args = cobra.ExactArgs(1)
		k.config = func(_ *cobra.Command, args []string) ([]byte, error) {
			return ioutil.ReadFile(args[0])
		}

		// This is kind of sneaky, but try to pickup the Kustomize root here
		k.root = os.Getenv("KUSTOMIZE_PLUGIN_CONFIG_ROOT")
	}
}

// WithPreRunE will setup the pre-run of the command to invoke the specified `preRunE` function before configuring the
// plugin itself (using a `nil` configuration byte array).
func WithPreRunE(preRunE func(cmd *cobra.Command, args []string) error) RunnerOption {
	return func(k *PluginRunner) {
		k.cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
			if err := preRunE(cmd, args); err != nil {
				return err
			}

			// Explicitly call through to the default implementation to invoke Configurable.Config
			return k.preRun(cmd, args)
		}
	}
}

func WithHashTransformer() RunnerOption {
	return func(k *PluginRunner) {
		k.addTransformerPlugin(builtin.NewHashTransformerPlugin(), nil)
		// TODO We can't just add this without having something to fix name references
	}
}

// WithTransformerFilenameFlag is used by transformers to allow input to come from a file instead of stdin.
// This should only be used with "Command" commands where the expectation is that we were invoked outside of Kustomize.
func WithTransformerFilenameFlag() RunnerOption {
	return func(k *PluginRunner) {
		type fileFlags struct {
			Filename string
		}
		f := &fileFlags{}
		k.cmd.Flags().StringVarP(&f.Filename, "filename", "f", "", "`file` that contains the manifests to transform")

		k.generate = func() (resmap.ResMap, error) {
			if f.Filename == "-" || f.Filename == "" {
				return k.newResMapFromStdin()
			}

			b, err := ioutil.ReadFile(f.Filename)
			if err != nil {
				return nil, err
			}
			return k.rf.NewResMapFromBytes(b)
		}
	}
}

// combineTransformFunc combines two transform functions into a single function. The second function must not be nil.
func combineTransformFunc(t1, t2 func(resmap.ResMap) error) func(resmap.ResMap) error {
	if t1 == nil {
		return t2
	}
	return func(m resmap.ResMap) error {
		if err := t1(m); err != nil {
			return err
		}
		return t2(m)
	}
}

// persistResourceOptions persists resource options using Kustomize annotations
func persistResourceOptions(m resmap.ResMap) error {
	for _, r := range m.Resources() {
		// Look for the Kustomize "id" annotation to determine if we should add other resource options
		annotations := r.GetAnnotations()
		if annotations["kustomize.config.k8s.io/id"] != "" {

			if r.Behavior() != types.BehaviorUnspecified {
				annotations["kustomize.config.k8s.io/behavior"] = r.Behavior().String()
			}

			if r.NeedHashSuffix() {
				annotations["kustomize.config.k8s.io/needs-hash"] = "true"
			}

			r.SetAnnotations(annotations)
		}
	}
	return nil
}