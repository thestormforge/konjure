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
	"github.com/carbonrelay/konjure/plugin/util"
	"github.com/spf13/cobra"
)

func NewJsonnetGeneratorExecPlugin() *cobra.Command {
	p := &plugin{}
	cmd := util.NewKustomizePluginRunner(p, util.WithConfigType("konjure.carbonrelay.com", "v1beta1", "JsonnetGenerator")).Command()
	return cmd
}

func NewJsonnetGeneratorCommand() *cobra.Command {
	p := &plugin{}
	f := &jsonnetFlags{}
	cmd := util.NewKustomizePluginRunner(p, f.withPreRun(p)).Command()
	cmd.Use = "jsonnet"
	cmd.Short = "Evaluate a Jsonnet program"
	cmd.Args = cobra.ExactArgs(1)

	cmd.Flags().BoolVarP(&f.execute, "exec", "e", false, "treat argument as code")
	cmd.Flags().StringToStringVarP(&f.externalStringVariables, "ext-str", "V", nil, "provide external variable as a string")
	cmd.Flags().StringToStringVar(&f.externalStringFileVariables, "ext-str-file", nil, "provide external variable as a string from the file")
	cmd.Flags().StringToStringVar(&f.externalCodeVariables, "ext-code", nil, "provide external variable as Jsonnet code")
	cmd.Flags().StringToStringVar(&f.externalCodeFileVariables, "ext-code-file", nil, "provide external variable as Jsonnet code from the file")
	cmd.Flags().StringToStringVarP(&f.topLevelStringArguments, "tla-str", "A", nil, "provide top-level argument as a string")
	cmd.Flags().StringToStringVar(&f.topLevelStringFileArguments, "tla-str-file", nil, "provide top-level argument as a string from the file")
	cmd.Flags().StringToStringVar(&f.topLevelCodeArguments, "tla-code", nil, "provide top-level argument as Jsonnet code")
	cmd.Flags().StringToStringVar(&f.topLevelCodeFileArguments, "tla-code-file", nil, "provide top-level argument as Jsonnet code from the file")
	cmd.Flags().StringArrayVarP(&p.JsonnetPath, "jpath", "J", nil, "specify an additional library search directory")

	return cmd
}

type jsonnetFlags struct {
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

func (f *jsonnetFlags) withPreRun(p *plugin) util.RunnerOption {
	return util.WithPreRunE(func(cmd *cobra.Command, args []string) error {
		// Exec
		if f.execute {
			p.Code = args[0]
		} else {
			p.Filename = args[0]
		}

		// External variables
		for k, v := range f.externalStringVariables {
			p.ExternalVariables = append(p.ExternalVariables, Parameter{Name: k, String: v})
		}
		for k, v := range f.externalStringFileVariables {
			p.ExternalVariables = append(p.ExternalVariables, Parameter{Name: k, StringFile: v})
		}
		for k, v := range f.externalCodeVariables {
			p.ExternalVariables = append(p.ExternalVariables, Parameter{Name: k, Code: v})
		}
		for k, v := range f.externalCodeFileVariables {
			p.ExternalVariables = append(p.ExternalVariables, Parameter{Name: k, CodeFile: v})
		}

		// Top-level arguments
		for k, v := range f.topLevelStringArguments {
			p.TopLevelArguments = append(p.TopLevelArguments, Parameter{Name: k, String: v})
		}
		for k, v := range f.topLevelStringFileArguments {
			p.TopLevelArguments = append(p.TopLevelArguments, Parameter{Name: k, StringFile: v})
		}
		for k, v := range f.topLevelCodeArguments {
			p.TopLevelArguments = append(p.TopLevelArguments, Parameter{Name: k, Code: v})
		}
		for k, v := range f.topLevelCodeFileArguments {
			p.TopLevelArguments = append(p.TopLevelArguments, Parameter{Name: k, CodeFile: v})
		}

		return nil
	})
}
