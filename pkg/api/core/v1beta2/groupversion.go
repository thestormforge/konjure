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
	Group      = "konjure.stormforge.io"
	Version    = "v1beta2"
	APIVersion = Group + "/" + Version
)

// AddTypeInfo overwrites the type information on the supplied instance if it is a pointer to one
// of our types.
func AddTypeInfo(obj interface{}) error {
	switch r := obj.(type) {
	case *Resource:
		r.APIVersion, r.Kind = APIVersion, "Resource"
	case *Helm:
		r.APIVersion, r.Kind = APIVersion, "Helm"
	case *Jsonnet:
		r.APIVersion, r.Kind = APIVersion, "Jsonnet"
	case *Kubernetes:
		r.APIVersion, r.Kind = APIVersion, "Kubernetes"
	case *Kustomize:
		r.APIVersion, r.Kind = APIVersion, "Kustomize"
	case *Secret:
		r.APIVersion, r.Kind = APIVersion, "Secret"
	case *Git:
		r.APIVersion, r.Kind = APIVersion, "Git"
	case *HTTP:
		r.APIVersion, r.Kind = APIVersion, "HTTP"
	case *File:
		r.APIVersion, r.Kind = APIVersion, "File"
	default:
		return fmt.Errorf("unknown type: %T", obj)
	}
	return nil
}

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
	if err := AddTypeInfo(obj); err != nil {
		return nil, err
	}

	data, err := yaml.Marshal(obj)
	if err != nil {
		return nil, err
	}

	return yaml.Parse(string(data))
}
