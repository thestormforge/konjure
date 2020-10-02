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
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/ifc"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/api/types"
	yyaml "sigs.k8s.io/yaml"
)

// ListResources attempts to list the resource references from the kustomization at the root loader.
func ListResources(ldr ifc.Loader) ([]*Resource, error) {
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
	resources := make([]*Resource, 0, len(k.Resources)+len(k.Bases))
	for _, target := range append(k.Resources, k.Bases...) {
		resources = append(resources, NewResource(target))
	}
	return resources, nil
}

// LoadResource returns a raw `ResMap` loaded from the resource target.
func LoadResource(fs filesys.FileSystem, resource *Resource) (resmap.ResMap, error) {
	return krusty.MakeKustomizer(fs, krusty.MakeDefaultOptions()).Run(resource.Target)
}

// LoadIdentifiers returns a mapping of resource identifiers to the reference they originated from. In other words,
// the resulting  map has an entry for the resources found in the supplied references.
func LoadIdentifiers(resources []*Resource) (map[resid.ResId]*Resource, error) {
	identifiers := make(map[resid.ResId]*Resource)
	fs := filesys.MakeFsOnDisk()
	for _, r := range resources {
		if r.Empty() {
			continue
		}

		rm, err := LoadResource(fs, r)
		if err != nil {
			return nil, err
		}

		for _, rmr := range rm.Resources() {
			identifiers[rmr.OrgId()] = r
		}
	}
	return identifiers, nil
}

// FindResource looks for resource information from the supplied map. If an exact match by resource original ID is
// not present, an attempt is made to match the resource independent of the namespace.
func FindResource(ids map[resid.ResId]*Resource, r *resource.Resource) *Resource {
	id := r.OrgId()
	res := ids[id]
	if res != nil {
		return res
	}

	// There are two possible reasons for this:
	// 1. The version information for the resource URL came up empty and we never loaded the actual resources
	// 2. The individual resource identifier was change via a transformation from the original

	// We cannot detect #2 running as an exec plugin (though it seems like the original identifier could
	// be propagated via annotation like the current identifier is).

	// We can guess (e.g. ignore the namespace) but we cannot get too aggressive because we don't know which
	// case we are looking at.

	for k, v := range ids {
		// If the original namespace was empty, allow a match excluding the new namespace
		if k.Namespace == "" && k.GvknEquals(id) {
			return v
		}
	}

	return nil
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
