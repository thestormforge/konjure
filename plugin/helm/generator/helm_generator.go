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
	"github.com/carbonrelay/konjure/internal/helm"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

// TODO Instead of "include tests" should we have a generic "exclude hooks" that defaults to the test hooks?

type plugin struct {
	h *resmap.PluginHelpers

	Helm         helm.Executor `json:"helm"`
	Repository   string        `json:"repo"`
	ReleaseName  string        `json:"releaseName"`
	Chart        string        `json:"chart"`
	Version      string        `json:"version"`
	Values       []helm.Value  `json:"values"`
	IncludeTests bool          `json:"includeTests"`
}

//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Generate() (resmap.ResMap, error) {
	// Make sure everything is configured
	p.Helm.Complete()

	// Initialize the client
	if err := p.Helm.Init(); err != nil {
		return nil, err
	}

	// Fetch the chart
	c, err := p.Helm.Fetch(p.Repository, p.Chart, p.Version)
	if err != nil {
		return nil, err
	}

	// Render the chart
	b, err := p.Helm.Template(c, p.ReleaseName, p.Values)
	if err != nil {
		return nil, err
	}

	// Convert to a resource map
	m, err := p.h.ResmapFactory().NewResMapFromBytes(b)
	if err != nil {
		return nil, err
	}

	// Strip chart tests
	if !p.IncludeTests {
		tests, err := m.Select(types.Selector{AnnotationSelector: "helm.sh/hook in (test-success, test-failure)"})
		if err != nil {
			return nil, err
		}
		for _, t := range tests {
			if err := m.Remove(t.OrgId()); err != nil {
				return nil, err
			}
		}
	}

	return m, nil
}
