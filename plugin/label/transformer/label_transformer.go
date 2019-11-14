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

	"sigs.k8s.io/kustomize/v3/pkg/gvk"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"
	"sigs.k8s.io/yaml"
)

// Kustomize #1567: API resources that assign specific meaning to an empty selector should not have
// a selector created by a label transformation or it impacts the correctness of the application. Any
// created selector would need to include the defaults documented by the API prior to inclusion of
// the new labels: it is easier to just not create the selector.

type plugin struct {
	ldr ifc.Loader
	rf  *resmap.Factory

	Labels     map[string]string  `json:"labels"`
	FieldSpecs []config.FieldSpec `json:"fieldSpecs,omitempty"`
}

var KustomizePlugin plugin

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) error {
	p.ldr = ldr
	p.rf = rf
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
	if len(p.FieldSpecs) == 0 {
		tc, _ := config.MakeDefaultConfig().Merge(config.MakeEmptyConfig())
		p.FieldSpecs = tc.CommonLabels
	}

	for _, r := range m.Resources() {
		for _, path := range p.FieldSpecs {
			if !r.OrgId().IsSelected(&path.Gvk) {
				continue
			}
			err := transformers.MutateField(r.Map(), path.PathSlice(), createIfNotPresent(r.GetGvk(), &path), p.addMap)
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
func createIfNotPresent(x gvk.Gvk, fs *config.FieldSpec) bool {

	// For replication controller, the default configuration contains an incorrect field specification
	if fs.Group == "" && fs.Version == "v1" && fs.Kind == "ReplicationController" && fs.Path == "spec/selector" {
		return false
	}

	if !fs.CreateIfNotPresent {
		return false
	}

	// TODO Is this the only other check we need, you should NEVER create selector/matchLabels...
	if fs.Path != "spec/selector/matchLabels" {
		return true
	}

	if x.Version != "v1beta1" {
		return true
	}

	// NOTE: Deployment is not explicitly documented as having this behavior, but `kubectl convert` does adjust it
	if x.Group == "apps" && x.Kind != "StatefulSet" && x.Kind != "Deployment" {
		return true
	}

	if x.Group == "extensions" && x.Kind != "DaemonSet" && x.Kind != "ReplicaSet" && x.Kind != "Deployment" {
		return true
	}

	return false
}
