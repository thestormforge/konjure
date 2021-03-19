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

package filters

import "sigs.k8s.io/kustomize/kyaml/yaml"

// SelectorFilter filters nodes based on label and/or annotation selectors.
type SelectorFilter struct {
	LabelSelector      string
	AnnotationSelector string
}

func (f *SelectorFilter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	// Optimization if both selectors are empty
	if f.LabelSelector == "" && f.AnnotationSelector == "" {
		return nodes, nil
	}

	// Iterate the nodes and apply the selectors
	result := make([]*yaml.RNode, 0, len(nodes))
	for _, n := range nodes {
		if f.LabelSelector != "" {
			if matches, err := n.MatchesLabelSelector(f.LabelSelector); err != nil {
				return nil, err
			} else if !matches {
				continue
			}
		}

		if f.AnnotationSelector != "" {
			if matches, err := n.MatchesAnnotationSelector(f.AnnotationSelector); err != nil {
				return nil, err
			} else if !matches {
				continue
			}
		}

		result = append(result, n)
	}

	return result, nil
}
