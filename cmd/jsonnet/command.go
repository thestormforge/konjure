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

package jsonnet

import (
	"bytes"
	"io"

	"github.com/spf13/cobra"
)

func NewJsonnetCommand() *cobra.Command {
	jc := &jsonnetCommand{
		options: NewJsonnetOptions(),
	}

	cmd := &cobra.Command{
		Use:    "jsonnet",
		Short:  "Evaluate a Jsonnet program",
		Args:   cobra.ExactArgs(1),
		PreRun: jc.preRun,
		RunE:   jc.run,
	}

	cmd.Flags().BoolVarP(&jc.execute, "exec", "e", false, "Treat argument as code")
	cmd.Flags().StringArrayVarP(&jc.options.JsonnetPath, "jpath", "J", nil, "Specify an additional library search directory")
	cmd.Flags().StringToStringVarP(&jc.externalStringVariables, "ext-str", "V", nil, "Provide external variable as a string")
	cmd.Flags().StringToStringVar(&jc.externalStringFileVariables, "ext-str-file", nil, "Provide external variable as a string from the file")
	cmd.Flags().StringToStringVar(&jc.externalCodeVariables, "ext-code", nil, "Provide external variable as Jsonnet code")
	cmd.Flags().StringToStringVar(&jc.externalCodeFileVariables, "ext-code-file", nil, "Provide external variable as Jsonnet code from the file")
	cmd.Flags().StringToStringVarP(&jc.topLevelStringArguments, "tla-str", "A", nil, "Provide top-level argument as a string")
	cmd.Flags().StringToStringVar(&jc.topLevelStringFileArguments, "tla-str-file", nil, "Provide top-level argument as a string from the file")
	cmd.Flags().StringToStringVar(&jc.topLevelCodeArguments, "tla-code", nil, "Provide top-level argument as Jsonnet code")
	cmd.Flags().StringToStringVar(&jc.topLevelCodeFileArguments, "tla-code-file", nil, "Provide top-level argument as Jsonnet code from the file")

	return cmd
}

type jsonnetCommand struct {
	options                     *JsonnetOptions
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

// Convert the compatibility options to real options
func (jc *jsonnetCommand) preRun(cmd *cobra.Command, args []string) {
	jc.options.Jsonnet.Complete()

	// Exec
	if jc.execute {
		jc.options.Code = args[0]
	} else {
		jc.options.Filename = args[0]
	}

	// External variables
	for k, v := range jc.externalStringVariables {
		jc.options.ExternalVariables = append(jc.options.ExternalVariables, Parameter{Name: k, String: v})
	}
	for k, v := range jc.externalStringFileVariables {
		jc.options.ExternalVariables = append(jc.options.ExternalVariables, Parameter{Name: k, StringFile: v})
	}
	for k, v := range jc.externalCodeVariables {
		jc.options.ExternalVariables = append(jc.options.ExternalVariables, Parameter{Name: k, Code: v})
	}
	for k, v := range jc.externalCodeFileVariables {
		jc.options.ExternalVariables = append(jc.options.ExternalVariables, Parameter{Name: k, CodeFile: v})
	}

	// Top-level arguments
	for k, v := range jc.topLevelStringArguments {
		jc.options.TopLevelArguments = append(jc.options.TopLevelArguments, Parameter{Name: k, String: v})
	}
	for k, v := range jc.topLevelStringFileArguments {
		jc.options.TopLevelArguments = append(jc.options.TopLevelArguments, Parameter{Name: k, StringFile: v})
	}
	for k, v := range jc.topLevelCodeArguments {
		jc.options.TopLevelArguments = append(jc.options.TopLevelArguments, Parameter{Name: k, Code: v})
	}
	for k, v := range jc.topLevelCodeFileArguments {
		jc.options.TopLevelArguments = append(jc.options.TopLevelArguments, Parameter{Name: k, CodeFile: v})
	}
}

func (jc *jsonnetCommand) run(cmd *cobra.Command, args []string) error {
	b, err := jc.options.Run(cmd.ErrOrStderr())
	if err != nil {
		return err
	}

	_, err = io.Copy(cmd.OutOrStdout(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	return nil
}
