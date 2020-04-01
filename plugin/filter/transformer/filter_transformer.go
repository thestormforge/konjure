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
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	h *resmap.PluginHelpers

	Selector types.Selector `json:"selector"`
}

//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
	// Build a mapping of resource identifier strings to keep
	keepers := make(map[string]bool, m.Size())

	// Match the keepers
	selected, err := m.Select(p.Selector)
	if err != nil {
		return err
	}
	for _, r := range selected {
		keepers[r.CurId().String()] = true
	}

	// Drop everything that didn't match
	for _, id := range m.AllIds() {
		if !keepers[id.String()] {
			if err := m.Remove(id); err != nil {
				return err
			}
		}
	}
	return nil
}
