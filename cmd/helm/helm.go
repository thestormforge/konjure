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

package helm

import (
	"sigs.k8s.io/kustomize/v3/pkg/types"
)

// HelmOptions is the configuration for expanding Helm charts
type HelmOptions struct {
	Helm         Helm        `json:"helm"`
	ReleaseName  string      `json:"releaseName"`
	Chart        string      `json:"chart"`
	Version      string      `json:"version"`
	Values       []HelmValue `json:"values"`
	IncludeTests bool        `json:"includeTests"`
}

// TODO Instead of "include tests" should we have a generic "exclude hooks" that defaults to the test hooks?

func NewHelmOptions() *HelmOptions {
	return &HelmOptions{}
}

func (o *HelmOptions) Run() ([]byte, error) {
	// Initialize the client
	if err := o.Helm.Init(); err != nil {
		return nil, err
	}

	// Fetch the chart
	c, err := o.Helm.Fetch(o.Chart, o.Version)
	if err != nil {
		return nil, err
	}

	// Render the chart
	m, err := o.Helm.Template(c, o.ReleaseName, o.Values)
	if err != nil {
		return nil, err
	}

	// Strip chart tests
	if !o.IncludeTests {
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

	return m.AsYaml()
}
