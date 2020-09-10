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
	"github.com/carbonrelay/konjure/internal/version"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filtersutil"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	h         *resmap.PluginHelpers
	resources []string

	LabelFieldSpecs []types.FieldSpec `json:"labelFieldSpecs,omitempty"`
}

//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
	// The resources field is only populated by the CLI, otherwise we should list the resources
	refs := p.resources
	if len(refs) == 0 {
		l, err := version.ListResources(p.h.Loader())
		if err != nil {
			return err
		}
		refs = l
	}

	// Default the label field specifications only if they were explicitly omitted
	if p.LabelFieldSpecs == nil {
		p.LabelFieldSpecs = version.DefaultLabelFieldSpecs()
	}

	// Load the origin mappings for all the resources
	// (it would be better if the YAML nodes or the ResMap had the URL associated with it)
	origins, err := version.LoadOrigins(refs)
	if err != nil {
		return err
	}

	for _, r := range m.Resources() {
		err := filtersutil.ApplyToJSON(version.Filter{
			Origin:          origins[r.CurId()],
			LabelFieldSpecs: p.LabelFieldSpecs,
		}, r)
		if err != nil {
			return err
		}
	}
	return nil
}
