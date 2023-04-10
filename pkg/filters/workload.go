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

import (
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// WorkloadFilter keeps only workload resources, i.e. those that directly or
// indirectly own pods. For this filter to work, all intermediate resources must
// be present and must have owner metadata specified.
type WorkloadFilter struct {
	// Flag indicating if this filter should act as a pass-through.
	Enabled bool
	// Flag indicating if this filter should allow auto-scaling resources to pass-through.
	CaptureAutoScaling bool
}

// Filter keeps all the workload resources.
func (f *WorkloadFilter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	if !f.Enabled {
		return nodes, nil
	}

	owners := make(map[yaml.ResourceIdentifier]*yaml.ResourceIdentifier, len(nodes))
	pods := make([]yaml.ResourceIdentifier, 0, len(nodes)/3)
	unscoped := make(map[yaml.ResourceIdentifier]struct{})
	for _, n := range nodes {
		md, err := n.GetMeta()
		if err != nil {
			return nil, err
		}

		id := md.GetIdentifier()

		// Keep track of pods
		if md.APIVersion == "v1" && md.Kind == "Pod" {
			pods = append(pods, id)
		}

		// Keep track of unscoped nodes
		if md.Namespace == "" {
			unscoped[id] = struct{}{}
		}

		// Index the owner with `controller=true`
		if err := n.PipeE(
			yaml.Lookup(yaml.MetadataField, "ownerReferences"),
			yaml.FilterFunc(func(object *yaml.RNode) (*yaml.RNode, error) {
				return nil, object.VisitElements(func(node *yaml.RNode) error {
					controller, _ := node.GetFieldValue("controller")
					if isController, ok := controller.(bool); !ok || !isController {
						return nil
					}

					owners[id] = &yaml.ResourceIdentifier{}
					return node.YNode().Decode(owners[id])
				})
			})); err != nil {
			return nil, err
		}
	}

	// Find all the distinct workloads by traversing up from the pods
	workloads := make(map[yaml.ResourceIdentifier]struct{}, len(nodes))
	for _, pod := range pods {
		workload := pod
		for {
			owner := owners[workload]
			if owner == nil {
				break
			}

			workload = *owner
			if _, ok := unscoped[workload]; !ok {
				workload.Namespace = pod.Namespace
			}
		}
		workloads[workload] = struct{}{}
	}

	// There were no pods found, assume everything we find with a pod template is a workload
	if len(pods) == 0 {
		for _, n := range nodes {
			err := n.PipeE(
				Has(yaml.LookupFirstMatch(yaml.ConventionalContainerPaths)),
				yaml.FilterFunc(func(object *yaml.RNode) (*yaml.RNode, error) {
					md, err := n.GetMeta()
					if err != nil {
						return nil, err
					}
					if owners[md.GetIdentifier()] == nil {
						workloads[md.GetIdentifier()] = struct{}{}
					}
					return nil, nil
				}))
			if err != nil {
				return nil, err
			}
		}
	}

	// Filter out the workloads
	result := make([]*yaml.RNode, 0, len(workloads))
	for _, n := range nodes {
		md, err := n.GetMeta()
		if err != nil {
			return nil, err
		}

		if strings.HasPrefix(md.APIVersion, "autoscaling/") && !f.CaptureAutoScaling {
			continue
		} else if _, isWorkload := workloads[md.GetIdentifier()]; !isWorkload {
			continue
		}

		result = append(result, n)
	}
	return result, nil
}
