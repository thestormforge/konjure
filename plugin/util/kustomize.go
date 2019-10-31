package util

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/k8sdeps/validator"
	"sigs.k8s.io/kustomize/v3/pkg/fs"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	fLdr "sigs.k8s.io/kustomize/v3/pkg/loader"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/types"
)

// ExecPluginGVK returns the GVK for the supplied executable plugin command; returns nil if the command is not an executable plugin
func ExecPluginGVK(cmd *cobra.Command) *metav1.GroupVersionKind {
	if cmd.Annotations["group"] == "" || cmd.Annotations["version"] == "" || cmd.Annotations["kind"] == "" {
		return nil
	}
	return &metav1.GroupVersionKind{
		Group:   cmd.Annotations["group"],
		Version: cmd.Annotations["version"],
		Kind:    cmd.Annotations["kind"],
	}
}

// ExecPluginPath returns the path to an executable plugin
func ExecPluginPath(pluginDir string, gvk *metav1.GroupVersionKind) string {
	return filepath.Join(pluginDir, gvk.Group, gvk.Version, strings.ToLower(gvk.Kind), gvk.Kind)
}

// KustomizePluginRunner is used to create Cobra commands that run Kustomize plugins
type KustomizePluginRunner struct {
	plugin interface{}
	cmd    *cobra.Command
	ldr    ifc.Loader
	rf     *resmap.Factory

	generate  func() (resmap.ResMap, error)
	transform func(resMap resmap.ResMap) error
}

// RunnerOption is an option that can be applied when creating a plugin runner
type RunnerOption func(*KustomizePluginRunner)

// NewKustomizePluginRunner creates a new runner for the supplied plugin and options
func NewKustomizePluginRunner(plugin interface{}, opts ...RunnerOption) *KustomizePluginRunner {
	k := &KustomizePluginRunner{
		plugin: plugin,
		cmd:    &cobra.Command{},
	}

	fSys := fs.MakeFsOnDisk()
	v := validator.NewKustValidator()
	k.ldr = fLdr.NewFileLoaderAtCwd(v, fSys)

	uf := kunstruct.NewKunstructuredFactoryImpl()
	pf := transformer.NewFactoryImpl()
	k.rf = resmap.NewFactory(resource.NewFactory(uf), pf)

	switch p := plugin.(type) {
	case resmap.Generator:
		k.generate = p.Generate

		k.cmd.PreRunE = k.defaultPreRun
		k.cmd.RunE = k.run
	case resmap.Transformer:
		k.generate = k.newResMapFromStdin
		k.transform = p.Transform

		k.cmd.PreRunE = k.defaultPreRun
		k.cmd.RunE = k.run
	}

	for _, opt := range opts {
		opt(k)
	}

	return k
}

// Command returns the Cobra command to run the Kustomize plugin
func (k *KustomizePluginRunner) Command() *cobra.Command {
	return k.cmd
}

// newResMapFromStdin reads stdin and parses it as a resource map
func (k *KustomizePluginRunner) newResMapFromStdin() (resmap.ResMap, error) {
	b, err := ioutil.ReadAll(k.cmd.InOrStdin())
	if err != nil {
		return nil, err
	}
	return k.rf.NewResMapFromBytes(b)
}

// defaultPreRun will invoke the configure method of the plugin with a nil configuration
func (k *KustomizePluginRunner) defaultPreRun(cmd *cobra.Command, args []string) error {
	if c, ok := k.plugin.(resmap.Configurable); ok {
		return c.Config(k.ldr, k.rf, nil)
	}
	return nil
}

// configFilePreRun will invoke the configure method of the plugin with the contents of file named in the first argument
func (k *KustomizePluginRunner) configFilePreRun(cmd *cobra.Command, args []string) error {
	// Read directly from the real file system since Kustomize can't know to tell us anything different
	config, err := ioutil.ReadFile(args[0])
	if err != nil {
		return err
	}

	if c, ok := k.plugin.(resmap.Configurable); ok {
		return c.Config(k.ldr, k.rf, config)
	}
	return nil
}

// run will actually run everything
func (k *KustomizePluginRunner) run(cmd *cobra.Command, args []string) error {
	m, err := k.generate()
	if err != nil {
		return err
	}

	if k.transform != nil {
		if err := k.transform(m); err != nil {
			return err
		}
	}

	b, err := m.AsYaml()
	if err != nil {
		return err
	}

	_, err = cmd.OutOrStdout().Write(b)
	return err
}

// WithConfigType will annotate the Cobra command with the GVK of the configuration schema; it will also setup the
// positional arguments and pre-run of the command to read a configuration file of the specified kind.
func WithConfigType(group, version, kind string) RunnerOption {
	return func(k *KustomizePluginRunner) {
		// Record the GVK information on the command
		k.cmd.Use = kind + " FILE"
		k.cmd.Short = fmt.Sprintf("Kustomize executable generator plugin for %s", kind)
		k.cmd.Annotations = map[string]string{
			"group":   group,
			"version": version,
			"kind":    kind,
		}

		// Require an argument for the configuration filename
		k.cmd.Args = cobra.ExactArgs(1)
		k.cmd.PreRunE = k.configFilePreRun
	}
}

// WithPreRunE will setup the pre-run of the command to invoke the specified `preRunE` function before configuring the
// plugin itself (using a `nil` configuration byte array).
func WithPreRunE(preRunE func(cmd *cobra.Command, args []string) error) RunnerOption {
	return func(k *KustomizePluginRunner) {
		k.cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
			if err := preRunE(cmd, args); err != nil {
				return err
			}

			// Explicitly call through to the default implementation to invoke Configurable.Config
			return k.defaultPreRun(cmd, args)
		}
	}
}

// WithAnnotationHashTransformer is used by generators to switch to downstream annotation based name suffix hashing.
// This should only be used from "ExecPlugin" commands where the expectation is that we were invoked by Kustomize.
func WithAnnotationHashTransformer(o *types.GeneratorOptions) RunnerOption {
	return func(k *KustomizePluginRunner) {
		origPreRunE := k.cmd.PreRunE
		k.cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
			// The original pre-run will unmarshal the configuration
			if origPreRunE != nil {
				if err := origPreRunE(cmd, args); err != nil {
					return err
				}
			}

			// If the name suffix hash is enabled, disable it and add the annotation instead
			if !o.DisableNameSuffixHash {
				o.DisableNameSuffixHash = true
				if o.Annotations == nil {
					o.Annotations = make(map[string]string, 1)
				}
				o.Annotations["kustomize.config.k8s.io/needs-hash"] = "true"
			}

			return nil
		}
	}
}

// WithTransformerFilenameFlag is used by transformers to allow input to come from a file instead of stdin.
// This should only be used with "Command" commands where the expectation is that we were invoked outside of Kustomize.
func WithTransformerFilenameFlag() RunnerOption {
	return func(k *KustomizePluginRunner) {
		type fileFlags struct {
			Filename string
		}
		f := &fileFlags{}
		k.cmd.Flags().StringVarP(&f.Filename, "filename", "f", "", "`file` that contains the manifests to transform")

		origRunE := k.cmd.RunE
		k.cmd.RunE = func(cmd *cobra.Command, args []string) error {
			if f.Filename != "-" && f.Filename != "" {
				k.generate = func() (resmap.ResMap, error) {
					b, err := ioutil.ReadFile(f.Filename)
					if err != nil {
						return nil, err
					}
					return k.rf.NewResMapFromBytes(b)
				}
			}

			if origRunE != nil {
				return origRunE(cmd, args)
			}
			return nil
		}
	}
}
