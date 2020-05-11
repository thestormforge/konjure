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
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	h *resmap.PluginHelpers

	Resources []string `json:"resources"`
}

//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Generate() (resmap.ResMap, error) {
	m := resmap.New()

	for _, res := range p.Resources {
		rm, err := p.h.ResmapFactory().FromFile(p.h.Loader(), res)
		if err != nil {
			return nil, err
		}
		if err := m.AbsorbAll(rm); err != nil {
			return nil, err
		}
	}

	return m, nil
}
