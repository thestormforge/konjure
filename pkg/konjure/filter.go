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
	"fmt"

	"github.com/thestormforge/konjure/internal/readers"
	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Filter replaces Konjure resources with the expanded resources they represent.
type Filter struct {
	// The number of times to recursively filter the resource list.
	Depth int
}

// Filter expands all of the Konjure resources using the configured executors.
func (f *Filter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	return f.filterToDepth(nodes, f.Depth)
}

// filterToDepth applies the expansion executors up to the specified depth (i.e. a File executor that produces a
// Kustomize resource would be at a depth of 2).
func (f *Filter) filterToDepth(nodes []*yaml.RNode, depth int) ([]*yaml.RNode, error) {
	if depth <= 0 {
		return nodes, nil
	}

	var cleaners readers.Cleaners
	defer func() {
		// TODO This should produce warnings, maybe the errors can be accumulated on the filer itself
		cleaners.CleanUp()
	}()

	var result []*yaml.RNode
	var depthNext int
	for _, n := range nodes {
		m, err := n.GetMeta()
		if err != nil {
			return nil, err
		}

		// Just include non-Konjure resources directly
		if m.APIVersion != konjurev1beta2.APIVersion {
			result = append(result, n)
			continue
		}

		// Only set the depth if we encounter a Konjure resource (which could expand into other Konjure resources)
		depthNext = depth - 1

		// Create a new typed object from the YAML
		obj, err := konjurev1beta2.NewForType(&m.TypeMeta)
		if err != nil {
			return nil, err
		}
		if err := n.YNode().Decode(obj); err != nil {
			return nil, err
		}

		// Create a reader
		r := ReaderFor(obj)
		if r == nil {
			return nil, fmt.Errorf("unable to read resources from type: %s", m.Kind)
		}

		// If a reader requires clean up, add it to the list
		if c, ok := r.(readers.Cleaner); ok {
			cleaners = append(cleaners, c)
		}

		// Accumulate additional resource nodes
		ns, err := r.Read()
		if err != nil {
			return nil, err
		}
		result = append(result, ns...)
	}

	return f.filterToDepth(result, depthNext)
}
