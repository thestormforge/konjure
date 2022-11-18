/*
Copyright 2022 GramLabs, Inc.

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

// WorkloadFilter keeps only workload resources, i.e. those that directly or
// indirectly own pods. For this filter to work, all intermediate resources must
// be present and must have owner metadata specified.
type WorkloadFilter struct {
	// Flag indicating if this filter should act as a pass-through.
	Enabled bool
}

// Filter keeps all the workload resources.
func (f *WorkloadFilter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	if !f.Enabled {
		return nodes, nil
	}

	// Index the owners
	owners := make(map[yaml.ResourceIdentifier][]yaml.ResourceIdentifier, len(nodes))
	for _, n := range nodes {
		md, err := n.GetMeta()
		if err != nil {
			return nil, err
		}
		id := md.GetIdentifier()
		if err = n.PipeE(
			yaml.Lookup(yaml.MetadataField, "ownerReferences"),
			yaml.FilterFunc(func(object *yaml.RNode) (*yaml.RNode, error) {
				return nil, object.VisitElements(func(node *yaml.RNode) error {
					owner := yaml.ResourceIdentifier{}
					if err := node.YNode().Decode(&owner); err != nil {
						return err
					}
					if owner.Namespace == "" {
						owner.Namespace = md.Namespace
					}
					owners[id] = append(owners[id], owner)
					return nil
				})
			})); err != nil {
			return nil, err
		}
	}
	var findOwners func(yaml.ResourceIdentifier) []yaml.ResourceIdentifier
	findOwners = func(id yaml.ResourceIdentifier) []yaml.ResourceIdentifier {
		if len(owners[id]) == 0 {
			return []yaml.ResourceIdentifier{id}
		}
		var result []yaml.ResourceIdentifier
		for _, owner := range owners[id] {
			result = append(result, findOwners(owner)...)
		}
		return result
	}

	// Find the owners of the pods
	workloads := make(map[yaml.ResourceIdentifier]struct{}, len(nodes))
	for _, n := range nodes {
		md, err := n.GetMeta()
		if err != nil {
			return nil, err
		}
		if md.APIVersion != "v1" || md.Kind != "Pod" {
			continue
		}

		for _, owner := range findOwners(md.GetIdentifier()) {
			workloads[owner] = struct{}{}
		}
	}

	// Take the full nodes
	result := make([]*yaml.RNode, 0, len(workloads))
	for _, n := range nodes {
		md, err := n.GetMeta()
		if err != nil {
			return nil, err
		}
		if _, ok := workloads[md.GetIdentifier()]; ok {
			result = append(result, n)
		}
	}

	return result, nil
}
