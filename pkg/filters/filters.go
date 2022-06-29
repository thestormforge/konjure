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
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"
	"sigs.k8s.io/kustomize/kyaml/yaml/walk"
)

// FilterOne is the opposite of kio.FilterAll, useful if you have a filter that
// is optimized for filtering batches of nodes, but you just need to call `Pipe`
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

// FilterAll is similar to `kio.FilterAll` except instead of evaluating for side
// effects, only the non-nil nodes returned by the filter are preserved.
func FilterAll(f yaml.Filter) kio.Filter {
	return kio.FilterFunc(func(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
		var result []*yaml.RNode
		for i := range nodes {
			n, err := f.Filter(nodes[i])
			if err != nil {
				return nil, err
			}
			if yaml.IsMissingOrNull(n) {
				continue
			}

			result = append(result, n)
		}
		return result, nil
	})
}

// Has is similar to `yaml.Tee` except it only produces a result if the supplied
// functions evaluate to a non-nil result.
func Has(functions ...yaml.Filter) yaml.Filter {
	return yaml.FilterFunc(func(rn *yaml.RNode) (*yaml.RNode, error) {
		n, err := rn.Pipe(functions...)
		if err != nil {
			return nil, err
		}
		if yaml.IsMissingOrNull(n) {
			return nil, nil
		}

		return rn, nil
	})
}

// Flatten never returns more than a single node, every other node is merged
// into that first node using the supplied schema
func Flatten(schema *spec.Schema) kio.Filter {
	return kio.FilterFunc(func(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
		// If there is 0 or 1 nodes, there is nothing for us to do
		if len(nodes) <= 1 {
			return nodes, nil
		}

		// Construct a 2-way merge walker using all the input nodes as sources
		w := walk.Walker{
			Visitor: merge2.Merger{},
			Sources: nodes,
		}
		if schema != nil {
			w.Schema = &openapi.ResourceSchema{Schema: schema}
		}

		// Return the result of the merge as a single node
		result, err := w.Walk()
		if err != nil || result == nil {
			return nil, err
		}
		return []*yaml.RNode{result}, nil
	})
}

// Pipeline wraps a KYAML pipeline but doesn't allow writers: instead the
// resulting resource nodes are returned directly. This is useful for applying
// filters to readers in memory. A pipeline can also be used as a reader in
// larger pipelines.
type Pipeline struct {
	Inputs                []kio.Reader
	Filters               []kio.Filter
	ContinueOnEmptyResult bool
}

// Execute this pipeline, returning the resulting resource nodes directly.
func (p *Pipeline) Read() ([]*yaml.RNode, error) {
	var result []*yaml.RNode

	pp := kio.Pipeline{
		Inputs:                p.Inputs,
		Filters:               p.Filters,
		ContinueOnEmptyResult: p.ContinueOnEmptyResult,
		Outputs: []kio.Writer{kio.WriterFunc(func(nodes []*yaml.RNode) error {
			result = nodes
			return nil
		})},
	}

	if err := pp.Execute(); err != nil {
		return nil, err
	}

	return result, nil
}
