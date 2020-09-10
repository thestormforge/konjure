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
	"net/url"
	"strings"

	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/filters/filtersutil"
	"sigs.k8s.io/kustomize/api/filters/fsslice"
	"sigs.k8s.io/kustomize/api/ifc"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	yyaml "sigs.k8s.io/yaml"
)

// ListResources attempts to list the resource references from the kustomization at the root loader.
func ListResources(ldr ifc.Loader) ([]string, error) {
	var b []byte
	var err error
	for _, fileName := range konfig.RecognizedKustomizationFileNames() {
		b, err = ldr.Load(fileName)
		if err == nil {
			break
		}
	}
	if len(b) == 0 || err != nil {
		return nil, err
	}
	k := struct {
		Resources []string `json:"resources"`
		Bases     []string `json:"bases"`
	}{}
	if err = yyaml.Unmarshal(b, &k); err != nil {
		return nil, err
	}
	return append(k.Resources, k.Bases...), nil
}

// LoadOrigins returns a mapping of resource identifiers to the reference they originated from. In other words,
// the resulting  map has an entry for every resource found in the supplied references.
func LoadOrigins(resources []string) (map[resid.ResId]string, error) {
	origins := make(map[resid.ResId]string)
	fs := filesys.MakeFsOnDisk()
	for _, r := range resources {
		// If we cannot use it later, don't bother evaluating the kustomization
		if ExtractInfo(r) == nil {
			continue
		}

		k := krusty.MakeKustomizer(fs, krusty.MakeDefaultOptions())
		rm, err := k.Run(r)
		if err != nil {
			return nil, err
		}

		for _, id := range rm.AllIds() {
			origins[id] = r
		}
	}
	return origins, nil
}

// DefaultLabelFieldSpecs returns the default set of field specifications for version labels.
func DefaultLabelFieldSpecs() types.FsSlice {
	return []types.FieldSpec{
		{
			Path:               "metadata/labels",
			CreateIfNotPresent: true,
		},
		{
			Path:               "spec/template/metadata/labels",
			CreateIfNotPresent: false,
			Gvk: resid.Gvk{
				Group: "apps",
			},
		},
		{
			Path:               "spec/jobTemplate/metadata/labels",
			CreateIfNotPresent: false,
			Gvk: resid.Gvk{
				Group: "batch",
				Kind:  "CronJob",
			},
		},
	}
}

// Filter is kyaml filter for adjusting the version of a resource based on it's origin reference.
type Filter struct {
	// The reference of the kustomization root where a resource originated
	Origin string
	// The field specs of the labels to update
	LabelFieldSpecs types.FsSlice
	// TODO Switch from "legacy" behavior to doing images based on field specs also?
}

func (f Filter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	origin := ExtractInfo(f.Origin)
	if origin == nil {
		return nodes, nil
	}

	return kio.FilterAll(yaml.FilterFunc(func(node *yaml.RNode) (*yaml.RNode, error) {
		// Apply the version number from the resource URL as a label
		if err := f.addVersionLabel(node, origin); err != nil {
			return nil, err
		}

		// Update the image names
		if err := f.setImage(node, origin); err != nil {
			return nil, err
		}

		return node, nil
	})).Filter(nodes)
}

func (f *Filter) addVersionLabel(node *yaml.RNode, origin *OriginInfo) error {
	if len(f.LabelFieldSpecs) == 0 || origin.Version == "" {
		return nil
	}
	return node.PipeE(fsslice.Filter{
		FsSlice:    f.LabelFieldSpecs,
		SetValue:   filtersutil.SetEntry("app.kubernetes.io/version", origin.Version, yaml.NodeTagString),
		CreateKind: yaml.MappingNode,
		CreateTag:  yaml.NodeTagString,
	})
}

func (f *Filter) setImage(node *yaml.RNode, origin *OriginInfo) error {
	if origin.ImageName == "" || origin.ImageTag == "" {
		return nil
	}
	if meta, err := node.GetMeta(); err != nil || meta.Kind == "CustomResourceDefinition" {
		return err // err will be nil for CRDs
	}
	return walkToImage(node, origin)
}

// OriginInfo represents the information extracted from a kustomization root reference.
type OriginInfo struct {
	// The version number to use when labelling resources
	Version string
	// The name of the image to adjust the tag of
	ImageName string
	// The new tag to apply to the image
	ImageTag string
}

// ExtractInfo attempts to extract origin information from a reference, it returns nil if it cannot extract information.
func ExtractInfo(origin string) *OriginInfo {
	u, err := url.Parse(origin)
	if err != nil {
		return nil
	}

	// Special handling for no-protocol GitHub references
	if strings.HasPrefix(u.Path, "github.com/") {
		u.Scheme = "git"
		u.Host = "github.com"
		u.Path = strings.TrimPrefix(u.Path, "github.com/")
	}

	// For now we are only taking ref parameters matching `v*`
	ref := u.Query().Get("ref")
	if ref == "" || !strings.HasPrefix(ref, "v") {
		return nil
	}

	info := &OriginInfo{
		Version: ref,
	}

	// If the host is GitHub, then we can guess the slug (`owner/repo`) is also the image name
	if u.Host == "github.com" {
		if p := strings.Split(strings.TrimPrefix(u.Path, "/"), "/"); len(p) >= 2 {
			info.ImageName = strings.ToLower(p[0] + "/" + p[1])
			info.ImageTag = strings.TrimPrefix(info.Version, "v")
		}
	}

	return info
}

// MatchesImage checks to see if the supplied image name is matched by this origin.
func (o *OriginInfo) MatchesImage(image string) bool {
	i := strings.LastIndexByte(image, '/')
	if i < 0 {
		i = 0
	}
	digest := strings.IndexByte(image[i:], '@')
	if digest > 0 {
		return o.ImageName == image[0:i+digest]
	}
	tag := strings.IndexByte(image[i:], ':')
	if tag > 0 {
		return o.ImageName == image[0:i+tag]
	}
	return image == o.ImageName
}

// Filter updates an image name.
func (o *OriginInfo) Filter(node *yaml.RNode) (*yaml.RNode, error) {
	if err := yaml.ErrorIfInvalid(node, yaml.ScalarNode); err != nil {
		return nil, err
	}
	if !o.MatchesImage(node.YNode().Value) {
		return node, nil
	}
	return node.Pipe(yaml.FieldSetter{StringValue: o.ImageName + ":" + o.ImageTag})
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
