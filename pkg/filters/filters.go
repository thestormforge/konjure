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
	"context"

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
		w := walk.Walker{Visitor: merge2.Merger{}}
		if schema != nil {
			w.Schema = &openapi.ResourceSchema{Schema: schema}
		}

		for i := len(nodes); i > 1; i-- {
			w.Sources = nodes[i-2 : i]
			if _, err := w.Walk(); err != nil {
				return nil, err
			}
			nodes = nodes[:i-1]
		}
		return nodes, nil
	})
}

// Pipeline is an alternate to the kio.Pipeline. This pipeline has the following differences:
// 1. The read/filter is separated so this pipeline can be used as a reader in another pipeline
// 2. This pipeline does not try to reconcile Kustomize annotations
// 3. This pipeline does not support callbacks
// 4. This pipeline implicitly clears empty annotations
type Pipeline struct {
	Inputs                []kio.Reader
	Filters               []kio.Filter
	Outputs               []kio.Writer
	ContinueOnEmptyResult bool
}

// Read evaluates the inputs and filters, ignoring the writers.
func (p *Pipeline) Read() ([]*yaml.RNode, error) {
	var result []*yaml.RNode

	// Read the inputs
	for _, input := range p.Inputs {
		nodes, err := input.Read()
		if err != nil {
			return nil, err
		}
		result = append(result, nodes...)
	}

	// Apply the filters
	for _, filter := range p.Filters {
		var err error
		result, err = filter.Filter(result)
		if err != nil {
			return nil, err
		}

		// Allow the filter loop to be stopped early if it goes empty
		if len(result) == 0 && !p.ContinueOnEmptyResult {
			break
		}
	}

	// Clear empty annotations on all nodes in the result
	for _, node := range result {
		if err := yaml.ClearEmptyAnnotations(node); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Execute reads and filters the nodes before sending them to the writers.
func (p *Pipeline) Execute() error {
	// Call Read to evaluate the Inputs and Filters
	nodes, err := p.Read()
	if err != nil {
		return err
	}

	// Check to see if the writers support empty node lists
	if len(nodes) == 0 && !p.ContinueOnEmptyResult {
		return nil
	}

	for _, output := range p.Outputs {
		if err := output.Write(nodes); err != nil {
			return err
		}
	}
	return nil
}

// ContextFilterFunc is a context-aware YAML filter function.
type ContextFilterFunc func(context.Context, *yaml.RNode) (*yaml.RNode, error)

// WithContext binds a context to a context filter function.
func WithContext(ctx context.Context, f ContextFilterFunc) yaml.Filter {
	return yaml.FilterFunc(func(node *yaml.RNode) (*yaml.RNode, error) {
		// Check the context error first, mainly for when this is wrapped in FilterAll
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return f(ctx, node)
	})
}
