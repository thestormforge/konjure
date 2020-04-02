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

// NewFilterTransformerCommand creates a new command for running filter from the CLI
func NewFilterTransformerCommand() *cobra.Command {
	p := &plugin{}
	cmd := kustomize.NewPluginRunner(p, kustomize.WithTransformerFilenameFlag())
	cmd.Use = "filter"
	cmd.Short = "Filter a stream of manifests"
	cmd.Long = "Possibly reduce the number of resources in a manifest stream"

	cmd.Flags().StringVar(&p.Selector.Group, "group", "", "`group` to match")
	cmd.Flags().StringVar(&p.Selector.Version, "version", "", "`version` to match")
	cmd.Flags().StringVar(&p.Selector.Kind, "kind", "", "`kind` to match")
	cmd.Flags().StringVarP(&p.Selector.Namespace, "namespace", "n", "", "`regex` of the namespaces to match")
	cmd.Flags().StringVar(&p.Selector.Name, "name", "", "`regex` of the names to match")
	cmd.Flags().StringVarP(&p.Selector.AnnotationSelector, "annotation-selector", "a", "", "`selector` to filter annotations on")
	cmd.Flags().StringVarP(&p.Selector.LabelSelector, "selector", "l", "", "`selector` to filter labels on")

	return cmd
}
