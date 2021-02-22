/*
Copyright 2021 GramLabs, Inc.

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

package konjure

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/thestormforge/konjure/internal/spec"
	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// NOTE: By design this code uses reflection instead of switch statements to isolate type changes to the api package

type Resource struct {
	// Instead of the "Resource" type, we just keep a single (internal use) string representing a single resource spec.
	str string

	Helm       *konjurev1beta2.Helm       `json:"helm,omitempty" yaml:"helm,omitempty"`
	Jsonnet    *konjurev1beta2.Jsonnet    `json:"jsonnet,omitempty" yaml:"jsonnet,omitempty"`
	Kubernetes *konjurev1beta2.Kubernetes `json:"kubernetes,omitempty" yaml:"kubernetes,omitempty"`
	Kustomize  *konjurev1beta2.Kustomize  `json:"kustomize,omitempty" yaml:"kustomize,omitempty"`
	Secret     *konjurev1beta2.Secret     `json:"secret,omitempty" yaml:"secret,omitempty"`
	Git        *konjurev1beta2.Git        `json:"git,omitempty" yaml:"git,omitempty"`
	HTTP       *konjurev1beta2.HTTP       `json:"http,omitempty" yaml:"http,omitempty"`
	File       *konjurev1beta2.File       `json:"file,omitempty" yaml:"file,omitempty"`
}

func (r *Resource) UnmarshalJSON(bytes []byte) error {
	if err := json.Unmarshal(bytes, &r.str); err == nil {
		rr, err := (&spec.Parser{}).Decode(r.str)
		if err != nil {
			return err
		}

		rv := reflect.Indirect(reflect.ValueOf(r))
		rrv := reflect.ValueOf(rr)
		for i := 0; i < rv.NumField(); i++ {
			if rv.Field(i).Type() == rrv.Type() {
				rv.Field(i).Set(rrv)
				return nil
			}
		}

		return fmt.Errorf("unknown resource type: %T", rr)
	}

	type rt *Resource
	return json.Unmarshal(bytes, rt(r))
}

func (r *Resource) MarshalJSON() ([]byte, error) {
	if r.str != "" {
		return json.Marshal(r.str)
	}

	type rt *Resource
	return json.Marshal(rt(r))
}

func (r *Resource) GetRNode() (*yaml.RNode, error) {
	rv := reflect.Indirect(reflect.ValueOf(r))
	for i := 0; i < rv.NumField(); i++ {
		if f := rv.Field(i); f.Kind() != reflect.String && !f.IsNil() {
			return konjurev1beta2.GetRNode(rv.Field(i).Interface())
		}
	}

	return nil, fmt.Errorf("resource is missing definition")
}

func (r *Resource) DeepCopyInto(rout *Resource) {
	rout.str = r.str

	rvin := reflect.Indirect(reflect.ValueOf(r))
	rvout := reflect.Indirect(reflect.ValueOf(rout))
	for i := 0; i < rvin.NumField(); i++ {
		if f := rvin.Field(i); f.Kind() != reflect.String && !f.IsNil() {
			rvout.Field(i).Set(reflect.New(f.Elem().Type()))
			rvout.Field(i).Elem().Set(f.Elem())
		}
	}
}

var _ kio.Reader = Resources{}

type Resources []Resource

func (rs Resources) Read() ([]*yaml.RNode, error) {
	result := make([]*yaml.RNode, 0, len(rs))
	for i := range rs {
		n, err := rs[i].GetRNode()
		if err != nil {
			return nil, err
		}
		result = append(result, n)
	}

	return result, nil
}
