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
	"github.com/carbonrelay/konjure/plugin/util"
	"github.com/spf13/cobra"
)

func NewLabelTransformerExecPlugin() *cobra.Command {
	p := &plugin{}
	cmd := util.NewKustomizePluginRunner(p, util.WithConfigType("konjure.carbonrelay.com", "v1beta1", "LabelTransformer"))
	return cmd
}

func NewLabelTransformerCommand() *cobra.Command {
	p := &plugin{}
	cmd := util.NewKustomizePluginRunner(p, util.WithTransformerFilenameFlag())
	cmd.Use = "label"
	cmd.Short = "Alternate to the Kustomize built-in LabelTransformer"
	cmd.Long = "Alternate to the built-in LabelTransformer, will be removed when selectors are no longer created"

	cmd.Flags().StringToStringVarP(&p.Labels, "label", "l", nil, "common `label`s to add")

	return cmd
}
