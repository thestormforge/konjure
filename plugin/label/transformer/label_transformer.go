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
	"fmt"
	"sort"

	"sigs.k8s.io/kustomize/api/konfig/builtinpluginconsts"
	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/transform"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

// Kustomize #1567: API resources that assign specific meaning to an empty selector should not have
// a selector created by a label transformation or it impacts the correctness of the application. Any
// created selector would need to include the defaults documented by the API prior to inclusion of
// the new labels: it is easier to just not create the selector.

type plugin struct {
	h *resmap.PluginHelpers

	Labels     map[string]string `json:"labels"`
	FieldSpecs []types.FieldSpec `json:"fieldSpecs,omitempty"`
}

var KustomizePlugin plugin

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
	if len(p.FieldSpecs) == 0 {
		type TransformerConfig struct {
			CommonLabels types.FsSlice `json:"commonLabels"`
		}
		tc := &TransformerConfig{}
		err := yaml.Unmarshal(builtinpluginconsts.GetDefaultFieldSpecs(), tc)
		if err != nil {
			return err
		}
		sort.Sort(tc.CommonLabels)
		p.FieldSpecs = tc.CommonLabels
	}

	for _, r := range m.Resources() {
		for _, path := range p.FieldSpecs {
			if !r.OrgId().IsSelected(&path.Gvk) {
				continue
			}
			err := transform.MutateField(r.Map(), path.PathSlice(), createIfNotPresent(r.GetGvk(), &path), p.addMap)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *plugin) addMap(in interface{}) (interface{}, error) {
	m, ok := in.(map[string]interface{})
	if in == nil {
		m = map[string]interface{}{}
	} else if !ok {
		return nil, fmt.Errorf("%#v is expected to be %T", in, m)
	}
	for k, v := range p.Labels {
		m[k] = v
	}
	return m, nil
}

// This is the additional check not present in the built-in transformer
func createIfNotPresent(x resid.Gvk, fs *types.FieldSpec) bool {
	// If the value is already false we do not need to worry about changing it
	if !fs.CreateIfNotPresent {
		return false
	}

	// For replication controller, the default configuration contains an incorrect field specification
	if fs.Group == "" && fs.Version == "v1" && fs.Kind == "ReplicationController" && fs.Path == "spec/selector" {
		return false
	}

	// We are only making changes to objects in the "apps" and "extensions" groups (we ignore the kind)
	if x.Group != "apps" && x.Group != "extensions" {
		return true
	}

	// Only adjust create value for match labels on v1beta1 resources
	if fs.Path != "spec/selector/matchLabels" || x.Version != "v1beta1" {
		return true
	}

	return false
}
