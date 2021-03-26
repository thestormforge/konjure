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
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// FilterOne is the opposite of kio.FilterAll, useful if you have a filter that
// is optimized for filtering batches of nodes but you just need to call `Pipe`
// on a single node.
func FilterOne(f kio.Filter) yaml.Filter {
	return yaml.FilterFunc(func(node *yaml.RNode) (*yaml.RNode, error) {
		nodes, err := f.Filter([]*yaml.RNode{node})
		if err != nil {
			return nil, err
		}

		if len(nodes) == 1 {
			return nodes[0], nil
		}

		return nil, nil
	})
}
