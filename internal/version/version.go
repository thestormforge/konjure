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
	"os"
	"regexp"
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

// Filter applies the version information extracted from the origin to the supplied document nodes.
func (f Filter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	origin := ExtractInfo(f.Origin)
	if origin == nil {
		return nodes, nil
	}
	// TODO The filter should have more image names we can add in

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
	if len(origin.ImageNames) == 0 || origin.ImageTag == "" {
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
	// The names of the images to adjust the tag of
	ImageNames []string
	// The new tag to apply to the image
	ImageTag string
}

var (
	versionTagRef = regexp.MustCompile(`^v([0-9]+\.[0-9]+\.[0-9]+)$`)
	hashRef       = regexp.MustCompile(`^([A-Fa-f0-9]{8})[A-Fa-f0-9]{32}$`)
)

// ExtractInfo attempts to extract origin information from a reference, it returns nil if it cannot extract information.
func ExtractInfo(origin string) *OriginInfo {
	if origin == "" {
		return nil
	}
	pwd, err := os.Getwd()
	if err != nil {
		return nil
	}
	force, u, err := detect(origin, pwd)
	if err != nil || force != "git" {
		return nil
	}

	info := &OriginInfo{}

	// Try to convert the ref to version and tag information
	ref := u.Query().Get("ref")
	if ref == "master" || ref == "" {
		info.ImageTag = "edge"
	} else if ms := versionTagRef.FindStringSubmatch(ref); ms != nil {
		info.Version = ref
		info.ImageTag = ms[1]
	} else if ms := hashRef.FindStringSubmatch(ref); ms != nil {
		info.ImageTag = "sha-" + ms[1]
	} else {
		info.ImageTag = strings.ReplaceAll(ref, "/", "-")
	}

	// Guess the image name matches the first two path segments
	if u.Host == "github.com" {
		if p := strings.Split(strings.TrimPrefix(u.Path, "/"), "/"); len(p) >= 2 {
			info.ImageNames = append(info.ImageNames, strings.TrimSuffix(strings.ToLower(p[0]+"/"+p[1]), ".git"))
		}
	}

	return info
}

// MatchesImage checks to see if the supplied image name is matched by this origin.
func (o *OriginInfo) MatchesImage(image string) (string, bool) {
	i := strings.LastIndexByte(image, '/')
	if i < 0 {
		i = 0
	}
	for _, imageName := range o.ImageNames {
		digest := strings.IndexByte(image[i:], '@')
		if digest > 0 {
			if imageName != image[0:i+digest] {
				continue
			}
			return imageName, true
		}

		tag := strings.IndexByte(image[i:], ':')
		if tag > 0 {
			if imageName != image[0:i+tag] {
				continue
			}
			return imageName, true
		}

		if imageName != image {
			continue
		}
		return imageName, true
	}
	return "", false
}

// Filter updates an image name.
func (o *OriginInfo) Filter(node *yaml.RNode) (*yaml.RNode, error) {
	if err := yaml.ErrorIfInvalid(node, yaml.ScalarNode); err != nil {
		return nil, err
	}
	if imageName, ok := o.MatchesImage(node.YNode().Value); ok {
		return node.Pipe(yaml.FieldSetter{StringValue: imageName + ":" + o.ImageTag})
	}
	return node, nil
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
