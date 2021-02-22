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

func NewJsonnetCommand() *cobra.Command {
	f := jsonnetFlags{}

	cmd := &cobra.Command{
		Use:    "jsonnet",
		Short:  "Evaluate a Jsonnet program",
		Args:   cobra.ExactArgs(1),
		PreRun: f.preRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			return kio.Pipeline{
				Inputs:  []kio.Reader{readers.NewJsonnetReader(&f.Jsonnet)},
				Outputs: []kio.Writer{&konjure.Writer{Writer: cmd.OutOrStdout()}},
			}.Execute()
		},
	}

	cmd.Flags().BoolVarP(&f.execute, "exec", "e", false, "treat argument as code")
	cmd.Flags().StringToStringVarP(&f.externalStringVariables, "ext-str", "V", nil, "provide external variable as a string")
	cmd.Flags().StringToStringVar(&f.externalStringFileVariables, "ext-str-file", nil, "provide external variable as a string from the file")
	cmd.Flags().StringToStringVar(&f.externalCodeVariables, "ext-code", nil, "provide external variable as Jsonnet code")
	cmd.Flags().StringToStringVar(&f.externalCodeFileVariables, "ext-code-file", nil, "provide external variable as Jsonnet code from the file")
	cmd.Flags().StringToStringVarP(&f.topLevelStringArguments, "tla-str", "A", nil, "provide top-level argument as a string")
	cmd.Flags().StringToStringVar(&f.topLevelStringFileArguments, "tla-str-file", nil, "provide top-level argument as a string from the file")
	cmd.Flags().StringToStringVar(&f.topLevelCodeArguments, "tla-code", nil, "provide top-level argument as Jsonnet code")
	cmd.Flags().StringToStringVar(&f.topLevelCodeFileArguments, "tla-code-file", nil, "provide top-level argument as Jsonnet code from the file")
	cmd.Flags().StringArrayVarP(&f.JsonnetPath, "jpath", "J", nil, "specify an additional library search directory")
	cmd.Flags().StringVar(&f.JsonnetBundlerPackageHome, "jsonnetpkg-home", "", "the directory used to cache packages in")
	cmd.Flags().BoolVar(&f.JsonnetBundlerRefresh, "jsonnetpkg-refresh", false, "force update dependencies")

	return cmd
}

type jsonnetFlags struct {
	konjurev1beta2.Jsonnet
	execute                     bool
	externalStringVariables     map[string]string
	externalStringFileVariables map[string]string
	externalCodeVariables       map[string]string
	externalCodeFileVariables   map[string]string
	topLevelStringArguments     map[string]string
	topLevelStringFileArguments map[string]string
	topLevelCodeArguments       map[string]string
	topLevelCodeFileArguments   map[string]string
}

func (f *jsonnetFlags) preRun(_ *cobra.Command, args []string) {
	if f.execute {
		f.Code = args[0]
	} else {
		f.Filename = args[0]
	}

	for k, v := range f.externalStringVariables {
		f.ExternalVariables = append(f.ExternalVariables, konjurev1beta2.JsonnetParameter{Name: k, String: v})
	}
	for k, v := range f.externalStringFileVariables {
		f.ExternalVariables = append(f.ExternalVariables, konjurev1beta2.JsonnetParameter{Name: k, StringFile: v})
	}
	for k, v := range f.externalCodeVariables {
		f.ExternalVariables = append(f.ExternalVariables, konjurev1beta2.JsonnetParameter{Name: k, Code: v})
	}
	for k, v := range f.externalCodeFileVariables {
		f.ExternalVariables = append(f.ExternalVariables, konjurev1beta2.JsonnetParameter{Name: k, CodeFile: v})
	}

	for k, v := range f.topLevelStringArguments {
		f.TopLevelArguments = append(f.TopLevelArguments, konjurev1beta2.JsonnetParameter{Name: k, String: v})
	}
	for k, v := range f.topLevelStringFileArguments {
		f.TopLevelArguments = append(f.TopLevelArguments, konjurev1beta2.JsonnetParameter{Name: k, StringFile: v})
	}
	for k, v := range f.topLevelCodeArguments {
		f.TopLevelArguments = append(f.TopLevelArguments, konjurev1beta2.JsonnetParameter{Name: k, Code: v})
	}
	for k, v := range f.topLevelCodeFileArguments {
		f.TopLevelArguments = append(f.TopLevelArguments, konjurev1beta2.JsonnetParameter{Name: k, CodeFile: v})
	}
}
