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

package generator

import (
	"github.com/carbonrelay/konjure/internal/helm"
	"github.com/carbonrelay/konjure/internal/kustomize"
	"github.com/spf13/cobra"
)

func NewHelmGeneratorExecPlugin() *cobra.Command {
	p := &plugin{}
	cmd := kustomize.NewPluginRunner(p, kustomize.WithConfigType("konjure.carbonrelay.com", "v1beta1", "HelmGenerator"))
	return cmd
}

func NewHelmGeneratorCommand() *cobra.Command {
	p := &plugin{}
	f := &helmFlags{}
	cmd := kustomize.NewPluginRunner(p, f.withPreRun(p))
	cmd.Use = "helm CHART"
	cmd.Short = "Inflate a Helm chart"
	cmd.Args = cobra.ExactArgs(1)

	// These flags match what real Helm has
	cmd.Flags().StringVar(&p.Repository, "repo", "", "repository `url` used to locate the chart")
	cmd.Flags().StringVarP(&p.ReleaseName, "name", "n", "release-name", "release `name`")
	cmd.Flags().StringVar(&p.Version, "version", "", "fetch a specific `version` of a chart; if empty, the latest version of the chart will be used")
	cmd.Flags().StringVar(&p.Helm.Home, "home", "", "override the location of your Helm config `directory`")
	cmd.Flags().StringToStringVar(&f.set, "set", nil, "set `value`s on the command line")
	cmd.Flags().StringToStringVar(&f.setFile, "set-file", nil, "set values from `file`s on the command line")
	cmd.Flags().StringToStringVar(&f.setString, "set-string", nil, "set string `value`s on the command line")
	cmd.Flags().StringArrayVarP(&f.values, "values", "f", nil, "specify values in a YAML `file`")

	// These flags are specific to our plugin
	cmd.Flags().BoolVar(&KustomizePlugin.IncludeTests, "include-tests", false, "do not remove resources labeled as test hooks")

	return cmd
}

// helmFlags is an extra structure for storing command line options. Unlike real Helm, we don't preserve order of set flags!
type helmFlags struct {
	set       map[string]string
	setFile   map[string]string
	setString map[string]string
	values    []string
}

// withPreRun will apply the stored flags to a plugin instance
func (f *helmFlags) withPreRun(p *plugin) kustomize.RunnerOption {
	return kustomize.WithPreRunE(func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			p.Chart = args[0]
		}

		for k, v := range f.set {
			p.Values = append(p.Values, helm.Value{Name: k, Value: v})
		}

		for k, v := range f.setFile {
			p.Values = append(p.Values, helm.Value{Name: k, Value: v, LoadFile: true})
		}

		for k, v := range f.setString {
			p.Values = append(p.Values, helm.Value{Name: k, Value: v, ForceString: true})
		}

		for _, valueFile := range f.values {
			p.Values = append(p.Values, helm.Value{File: valueFile})
		}

		return nil
	})
}
