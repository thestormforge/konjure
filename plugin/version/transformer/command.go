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

package transformer

import (
	"github.com/carbonrelay/konjure/internal/kustomize"
	"github.com/spf13/cobra"
)

// NewVersionTransformerExecPlugin creates a new command for running label as an executable plugin
func NewVersionTransformerExecPlugin() *cobra.Command {
	p := &plugin{}
	cmd := kustomize.NewPluginRunner(p, kustomize.WithConfigType("konjure.carbonrelay.com", "v1beta1", "VersionTransformer"))
	return cmd
}

// NewVersionTransformerCommand creates a new command for running label from the CLI
func NewVersionTransformerCommand() *cobra.Command {
	p := &plugin{}
	cmd := kustomize.NewPluginRunner(p, kustomize.WithTransformerFilenameFlag())
	cmd.Use = "version"
	cmd.Short = "Transformations for more consistent versioning"
	cmd.Long = "Related transformations that make versioning off-the-shelf application more smooth."

	cmd.Flags().StringArrayVar(&p.resources, "resources", nil, "resources")

	return cmd
}
