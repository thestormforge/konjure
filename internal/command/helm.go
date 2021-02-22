/*
Copyright 2021 GramLabs, Inc.

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

package command

import (
	"github.com/spf13/cobra"
	"github.com/thestormforge/konjure/internal/readers"
	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"github.com/thestormforge/konjure/pkg/konjure"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func NewHelmCommand() *cobra.Command {
	f := helmFlags{}

	cmd := &cobra.Command{
		Use:    "helm CHART",
		Short:  "Inflate a Helm chart",
		Args:   cobra.ExactArgs(1),
		PreRun: f.preRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			return kio.Pipeline{
				Inputs:  []kio.Reader{readers.NewHelmReader(&f.Helm)},
				Outputs: []kio.Writer{&konjure.Writer{Writer: cmd.OutOrStdout()}},
			}.Execute()
		},
	}

	// These flags match what real Helm has
	cmd.Flags().StringVar(&f.Helm.Repository, "repo", "", "repository `url` used to locate the chart")
	cmd.Flags().StringVar(&f.Helm.ReleaseName, "name", "RELEASE-NAME", "release `name`")
	cmd.Flags().StringVarP(&f.Helm.ReleaseNamespace, "namespace", "n", "default", "release `namespace`")
	cmd.Flags().StringVar(&f.Helm.Version, "version", "", "fetch a specific `version` of a chart; if empty, the latest version of the chart will be used")
	cmd.Flags().StringVar(&f.Helm.Helm.RepositoryCache, "repository-cache", "", "override the `directory` of your cached Helm repository index")
	cmd.Flags().StringToStringVar(&f.set, "set", nil, "set `value`s on the command line")
	cmd.Flags().StringToStringVar(&f.setFile, "set-file", nil, "set values from `file`s on the command line")
	cmd.Flags().StringToStringVar(&f.setString, "set-string", nil, "set string `value`s on the command line")
	cmd.Flags().StringArrayVarP(&f.values, "values", "f", nil, "specify values in a YAML `file`")

	// These flags are specific to our plugin
	cmd.Flags().BoolVar(&f.Helm.IncludeTests, "include-tests", false, "do not remove resources labeled as test hooks")

	return cmd
}

// helmFlags is an extra structure for storing command line options. Unlike real Helm, we don't preserve order of set flags!
type helmFlags struct {
	konjurev1beta2.Helm
	set       map[string]string
	setFile   map[string]string
	setString map[string]string
	values    []string
}

func (f *helmFlags) preRun(_ *cobra.Command, args []string) {
	if len(args) > 0 {
		f.Chart = args[0]
	}

	for k, v := range f.set {
		f.Values = append(f.Values, konjurev1beta2.HelmValue{Name: k, Value: v})
	}

	for k, v := range f.setFile {
		f.Values = append(f.Values, konjurev1beta2.HelmValue{Name: k, Value: v, LoadFile: true})
	}

	for k, v := range f.setString {
		f.Values = append(f.Values, konjurev1beta2.HelmValue{Name: k, Value: v, ForceString: true})
	}

	for _, valueFile := range f.values {
		f.Values = append(f.Values, konjurev1beta2.HelmValue{File: valueFile})
	}
}
