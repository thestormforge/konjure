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
	"github.com/thestormforge/konjure/internal/readers"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Filter replaces Konjure resources with the expanded resources they represent.
type Filter struct {
	// The number of times to recursively filter the resource list.
	Depth int
}

// Filter expands all of the Konjure resources using the configured executors.
func (f *Filter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	return (&readers.ReadersFilter{Depth: f.Depth}).Filter(nodes)
}
