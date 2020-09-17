/*
Copyright 2020 GramLabs, Inc.

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

package version

import (
	"sigs.k8s.io/kustomize/api/filters/filtersutil"
	"sigs.k8s.io/kustomize/api/filters/fsslice"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Filter is kyaml filter for adjusting the version of a resource based on it's origin reference.
type Filter struct {
	// The reference of the kustomization root where a resource originated
	Resource *Resource
	// The field specs of the labels to update
	LabelFieldSpecs types.FsSlice
}

// Filter applies the version information extracted from the origin to the supplied document nodes.
func (f Filter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	if f.Resource.Empty() {
		return nodes, nil
	}

	return kio.FilterAll(yaml.FilterFunc(func(node *yaml.RNode) (*yaml.RNode, error) {
		// Apply the version label
		if err := f.addVersionLabel(node); err != nil {
			return nil, err
		}

		// Update the image names
		if err := f.setImages(node); err != nil {
			return nil, err
		}

		return node, nil
	})).Filter(nodes)
}

func (f *Filter) addVersionLabel(node *yaml.RNode) error {
	if f.Resource.Version == "" || len(f.LabelFieldSpecs) == 0 {
		return nil
	}

	return node.PipeE(fsslice.Filter{
		FsSlice:    f.LabelFieldSpecs,
		SetValue:   filtersutil.SetEntry("app.kubernetes.io/version", f.Resource.Version, yaml.NodeTagString),
		CreateKind: yaml.MappingNode,
		CreateTag:  yaml.NodeTagString,
	})
}

func (f *Filter) setImages(node *yaml.RNode) error {
	if len(f.Resource.ImageNames) == 0 || f.Resource.ImageTag == "" {
		return nil
	}
	if meta, err := node.GetMeta(); err != nil || meta.Kind == "CustomResourceDefinition" {
		return err // err will be nil for CRDs
	}

	// TODO Switch from the "legacy" behavior to doing images based on field specs also?

	return walkToImage(node, yaml.FilterFunc(func(node *yaml.RNode) (*yaml.RNode, error) {
		if err := yaml.ErrorIfInvalid(node, yaml.ScalarNode); err != nil {
			return nil, err
		}

		image := f.Resource.MatchedImageName(node.YNode().Value)
		if image == "" {
			return node, nil
		}

		image += ":" + f.Resource.ImageTag
		return node.Pipe(yaml.FieldSetter{StringValue: image})
	}))
}

// walkToImage is a helper to descend down to the image specification of a container and apply the supplied filter.
// NOTE: This uses the "legacy" behavior of updating `**/container[]/image` or `**/initContainer[]/image`
func walkToImage(node *yaml.RNode, imageFilter yaml.Filter) error {
	switch node.YNode().Kind {
	case yaml.MappingNode:
		return node.VisitFields(func(node *yaml.MapNode) error {
			if err := walkToImage(node.Value, imageFilter); err != nil {
				return err
			}

			// Only continue processing container lists
			key := node.Key.YNode().Value
			if (key != "containers" && key != "initContainers") || node.Value.YNode().Kind != yaml.SequenceNode {
				return nil
			}

			return node.Value.VisitElements(func(node *yaml.RNode) error {
				return node.PipeE(yaml.Get("image"), imageFilter)
			})
		})
	case yaml.SequenceNode:
		return node.VisitElements(func(node *yaml.RNode) error {
			return walkToImage(node, imageFilter)
		})
	default:
		return nil
	}
}
