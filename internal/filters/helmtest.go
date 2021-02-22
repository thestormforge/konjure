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

import (
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type HelmTestFilter struct{}

func (h *HelmTestFilter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	result := make([]*yaml.RNode, 0, len(nodes))
	for _, n := range nodes {
		isTest, err := n.MatchesAnnotationSelector("helm.sh/hook in (test-success, test-failure)")
		if err != nil {
			return nil, err
		}
		if !isTest {
			result = append(result, n)
		}
	}

	return result, nil
}
