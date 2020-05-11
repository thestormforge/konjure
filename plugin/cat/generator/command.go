/*
Copyright 2020 GramLabs, Inc.

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
	"github.com/carbonrelay/konjure/internal/kustomize"
	"github.com/spf13/cobra"
)

// NewCatGeneratorExecPlugin creates a new command for running cat as an executable plugin
func NewCatGeneratorExecPlugin() *cobra.Command {
	p := &plugin{}
	cmd := kustomize.NewPluginRunner(p, kustomize.WithConfigType("konjure.carbonrelay.com", "v1beta1", "CatGenerator"))
	return cmd
}

// NewCatGeneratorCommand creates a new command for running cat from the CLI
func NewCatGeneratorCommand() *cobra.Command {
	p := &plugin{}
	cmd := kustomize.NewPluginRunner(p, kustomize.WithPreRunE(func(cmd *cobra.Command, args []string) error {
		p.Resources = args
		return nil
	}))
	cmd.Use = "cat RESOURCE..."
	cmd.Short = "Concatenate and print resources"
	cmd.Args = cobra.MinimumNArgs(1)

	return cmd
}
