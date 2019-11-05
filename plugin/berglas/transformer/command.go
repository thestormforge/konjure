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

package transformer

import (
	"github.com/carbonrelay/konjure/internal/kustomize"
	"github.com/spf13/cobra"
)

// NewBerglasTransformerExecPlugin creates a new command for running Berglas as an executable plugin
func NewBerglasTransformerExecPlugin() *cobra.Command {
	p := &plugin{}
	cmd := kustomize.NewPluginRunner(p, kustomize.WithConfigType("konjure.carbonrelay.com", "v1beta1", "BerglasTransformer"))
	return cmd
}

// NewBerglasCommand creates a new command for running Berglas from the CLI
func NewBerglasTransformerCommand() *cobra.Command {
	p := &plugin{}
	cmd := kustomize.NewPluginRunner(p, kustomize.WithTransformerFilenameFlag())
	cmd.Use = "transform"
	cmd.Short = "Transform an application leveraging Berglas references"

	//cmd.Flags().BoolVar(&p.GenerateSecrets, "secrets", false, "perform transformation using secrets")

	return cmd
}
