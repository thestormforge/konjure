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

package v1beta2

import (
	"fmt"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	// Group name for Konjure resources.
	Group = "konjure.stormforge.io"
	// Version is the current version number for Konjure resources.
	Version = "v1beta2"
	// APIVersion is the combined group and version string.
	APIVersion = Group + "/" + Version
)

// NewForType returns a new instance of the typed object identified by the supplied type metadata.
func NewForType(t *yaml.TypeMeta) (interface{}, error) {
	if t.APIVersion != APIVersion {
		return nil, fmt.Errorf("unknown API version: %s", t.APIVersion)
	}

	var result interface{}
	switch t.Kind {
	case "Resource":
		result = new(Resource)
	case "Helm":
		result = new(Helm)
	case "Jsonnet":
		result = new(Jsonnet)
	case "Kubernetes":
		result = new(Kubernetes)
	case "Kustomize":
		result = new(Kustomize)
	case "Secret":
		result = new(Secret)
	case "Git":
		result = new(Git)
	case "HTTP":
		result = new(HTTP)
	case "File":
		result = new(File)
	default:
		return nil, fmt.Errorf("unknown kind: %s", t.Kind)
	}

	return result, nil
}

// GetRNode converts the supplied object to a resource node.
func GetRNode(obj interface{}) (*yaml.RNode, error) {
	m := &yaml.ResourceMeta{TypeMeta: yaml.TypeMeta{APIVersion: APIVersion}}
	var node interface{}
	switch s := obj.(type) {
	case *Resource:
		m.Kind = "Resource"
		node = struct {
			Meta *yaml.ResourceMeta `yaml:",inline"`
			Spec *Resource          `yaml:",inline"`
		}{Meta: m, Spec: s}
	case *Helm:
		m.Kind = "Helm"
		node = struct {
			Meta *yaml.ResourceMeta `yaml:",inline"`
			Spec *Helm              `yaml:",inline"`
		}{Meta: m, Spec: s}
	case *Jsonnet:
		m.Kind = "Jsonnet"
		node = struct {
			Meta *yaml.ResourceMeta `yaml:",inline"`
			Spec *Jsonnet           `yaml:",inline"`
		}{Meta: m, Spec: s}
	case *Kubernetes:
		m.Kind = "Kubernetes"
		node = struct {
			Meta *yaml.ResourceMeta `yaml:",inline"`
			Spec *Kubernetes        `yaml:",inline"`
		}{Meta: m, Spec: s}
	case *Kustomize:
		m.Kind = "Kustomize"
		node = struct {
			Meta *yaml.ResourceMeta `yaml:",inline"`
			Spec *Kustomize         `yaml:",inline"`
		}{Meta: m, Spec: s}
	case *Secret:
		m.Kind = "Secret"
		node = struct {
			Meta *yaml.ResourceMeta `yaml:",inline"`
			Spec *Secret            `yaml:",inline"`
		}{Meta: m, Spec: s}
	case *Git:
		m.Kind = "Git"
		node = struct {
			Meta *yaml.ResourceMeta `yaml:",inline"`
			Spec *Git               `yaml:",inline"`
		}{Meta: m, Spec: s}
	case *HTTP:
		m.Kind = "HTTP"
		node = struct {
			Meta *yaml.ResourceMeta `yaml:",inline"`
			Spec *HTTP              `yaml:",inline"`
		}{Meta: m, Spec: s}
	case *File:
		m.Kind = "File"
		node = struct {
			Meta *yaml.ResourceMeta `yaml:",inline"`
			Spec *File              `yaml:",inline"`
		}{Meta: m, Spec: s}
	default:
		return nil, fmt.Errorf("unknown type: %T", obj)
	}

	data, err := yaml.Marshal(node)
	if err != nil {
		return nil, err
	}

	return yaml.Parse(string(data))
}
