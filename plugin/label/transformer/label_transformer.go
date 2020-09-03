/*
Copyright 2019 GramLabs, Inc.

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

package transformer

import (
	"sort"

	"sigs.k8s.io/kustomize/api/filters/fieldspec"
	"sigs.k8s.io/kustomize/api/filters/filtersutil"
	"sigs.k8s.io/kustomize/api/konfig/builtinpluginconsts"
	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	kyamlfiltersutil "sigs.k8s.io/kustomize/kyaml/filtersutil"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kyamlyaml "sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/yaml"
)

// Kustomize #1567: API resources that assign specific meaning to an empty selector should not have
// a selector created by a label transformation or it impacts the correctness of the application. Any
// created selector would need to include the defaults documented by the API prior to inclusion of
// the new labels: it is easier to just not create the selector.

type plugin struct {
	h *resmap.PluginHelpers

	Labels     map[string]string `json:"labels"`
	FieldSpecs []types.FieldSpec `json:"fieldSpecs,omitempty"`
}

//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Transform(m resmap.ResMap) error {
	if len(p.FieldSpecs) == 0 {
		type TransformerConfig struct {
			CommonLabels types.FsSlice `json:"commonLabels"`
		}
		tc := &TransformerConfig{}
		err := yaml.Unmarshal(builtinpluginconsts.GetDefaultFieldSpecs(), tc)
		if err != nil {
			return err
		}
		sort.Sort(tc.CommonLabels)
		p.FieldSpecs = tc.CommonLabels
	}

	for _, r := range m.Resources() {
		if err := kyamlfiltersutil.ApplyToJSON(p, r); err != nil {
			return err
		}
	}
	return nil
}

func (p *plugin) Filter(nodes []*kyamlyaml.RNode) ([]*kyamlyaml.RNode, error) {
	keys := filtersutil.SortedMapKeys(p.Labels)
	return kio.FilterAll(kyamlyaml.FilterFunc(func(node *kyamlyaml.RNode) (*kyamlyaml.RNode, error) {
		for _, k := range keys {
			err := node.PipeE(p.setLabel(k, p.Labels[k]))
			if err != nil {
				return nil, err
			}
		}
		return node, nil
	})).Filter(nodes)
}

func (p *plugin) setLabel(key, value string) kyamlyaml.Filter {
	return kyamlyaml.FilterFunc(func(node *kyamlyaml.RNode) (*kyamlyaml.RNode, error) {
		rm, err := node.GetMeta()
		if err != nil {
			return nil, err
		}
		group, version := resid.ParseGroupVersion(rm.APIVersion)

		for _, fs := range p.FieldSpecs {
			// Override the field spec on each iteration
			fs.CreateIfNotPresent = createIfNotPresent(group, version, &fs)
			return (&fieldspec.Filter{
				FieldSpec:  fs,
				SetValue:   filtersutil.SetEntry(key, value, kyamlyaml.NodeTagString),
				CreateKind: kyamlyaml.MappingNode,
				CreateTag:  kyamlyaml.NodeTagMap,
			}).Filter(node)
		}
		return node, nil
	})
}

// This is the additional check not present in the built-in transformer
func createIfNotPresent(group, version string, fs *types.FieldSpec) bool {
	// If the value is already false we do not need to worry about changing it
	if !fs.CreateIfNotPresent {
		return false
	}

	// For replication controller, the default configuration contains an incorrect field specification
	if fs.Group == "" && fs.Version == "v1" && fs.Kind == "ReplicationController" && fs.Path == "spec/selector" {
		return false
	}

	// We are only making changes to objects in the "apps" and "extensions" groups (we ignore the kind)
	if group != "apps" && group != "extensions" {
		return true
	}

	// Only adjust create value for match labels on v1beta1 resources
	if fs.Path != "spec/selector/matchLabels" || version != "v1beta1" {
		return true
	}

	return false
}
