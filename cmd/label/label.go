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

package label

import (
	"fmt"

	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/pkg/gvk"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/transformers"
	"sigs.k8s.io/kustomize/v3/pkg/transformers/config"
)

// Kustomize #1567: API resources that assign specific meaning to an empty selector should not have
// a selector created by a label transformation or it impacts the correctness of the application. Any
// created selector would need to include the defaults documented by the API prior to inclusion of
// the new labels: it is easier to just not create the selector.

// LabelOptions is the configuration for executing label transformations
type LabelOptions struct {
	Labels     map[string]string  `json:"labels"`
	FieldSpecs []config.FieldSpec `json:"fieldSpecs,omitempty"`
}

func NewLabelOptions() *LabelOptions {
	return &LabelOptions{}
}

func (o *LabelOptions) Run(in []byte) ([]byte, error) {
	if len(o.FieldSpecs) == 0 {
		tc, _ := config.MakeDefaultConfig().Merge(config.MakeEmptyConfig())
		o.FieldSpecs = tc.CommonLabels
	}

	rf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), nil)
	m, err := rf.NewResMapFromBytes(in)
	if err != nil {
		return nil, err
	}
	if err := o.Transform(m); err != nil {
		return nil, err
	}
	return m.AsYaml()
}

func (o *LabelOptions) Transform(m resmap.ResMap) error {
	for _, r := range m.Resources() {
		for _, path := range o.FieldSpecs {
			if !r.OrgId().IsSelected(&path.Gvk) {
				continue
			}
			err := transformers.MutateField(r.Map(), path.PathSlice(), createIfNotPresent(r.GetGvk(), &path), o.addMap)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (o *LabelOptions) addMap(in interface{}) (interface{}, error) {
	m, ok := in.(map[string]interface{})
	if in == nil {
		m = map[string]interface{}{}
	} else if !ok {
		return nil, fmt.Errorf("%#v is expected to be %T", in, m)
	}
	for k, v := range o.Labels {
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

	if x.Group == "extensions" && x.Kind != "DaemonSet" && x.Kind != "ReplicaSet" {
		return true
	}

	return false
}
