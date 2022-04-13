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

package readers

import (
	"fmt"

	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Filter is a KYAML Filter that maps Konjure resource specifications to
// KYAML Readers, then reads and flattens the resulting RNodes into the final
// result. Due to the recursive nature of this filter, the depth (number of
// allowed recursive iterations) must be specified; the default value of 0 is
// effectively a no-op.
type Filter struct {
	// The number of iterations to perform when expanding Konjure resources.
	Depth int
	// Configuration options for the readers.
	ReaderOptions []Option
}

// Filter expands all the Konjure resources using the configured executors.
func (f *Filter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	var err error

	// Recursively expand the nodes to the specified depth
	nodes, err = f.expandToDepth(nodes, f.Depth)
	if err != nil {
		return nil, err
	}

	return nodes, nil
}

// expandToDepth applies the expansion executors up to the specified depth (i.e. a File executor that produces a
// Kustomize resource would be at a depth of 2).
func (f *Filter) expandToDepth(nodes []*yaml.RNode, depth int) ([]*yaml.RNode, error) {
	if depth <= 0 {
		return nodes, nil
	}

	var opts []Option
	opts = append(opts, f.ReaderOptions...)

	// Create a new cleaner for this iteration
	cleanOpt, doClean := clean()
	opts = append(opts, cleanOpt)
	defer doClean()

	// Process each of the nodes
	result := make([]*yaml.RNode, 0, len(nodes))
	done := true
	for _, n := range nodes {
		r, err := f.expand(n)
		if err != nil {
			return nil, err
		}

		for _, opt := range opts {
			r = opt(r)
		}

		expanded, err := r.Read()
		if err != nil {
			return nil, err
		}

		if len(expanded) == 1 && expanded[0] != n {
			done = false
		}

		result = append(result, expanded...)
	}

	// Perform another iteration if any of the nodes changed
	if !done {
		return f.expandToDepth(result, depth-1)
	}
	return result, nil
}

// expand returns a reader which can expand the supplied node. If the supplied node
// cannot be expanded, the resulting reader will only produce that node.
func (f *Filter) expand(node *yaml.RNode) (kio.Reader, error) {
	m, err := node.GetMeta()
	if err != nil {
		return nil, err
	}

	switch {

	case m.APIVersion == konjurev1beta2.APIVersion:
		// Unmarshal the typed Konjure resource and create a reader from it
		res, err := konjurev1beta2.NewForType(&m.TypeMeta)
		if err != nil {
			return nil, err
		}
		if err := node.YNode().Decode(res); err != nil {
			return nil, err
		}
		r := New(res)
		if r == nil {
			return nil, fmt.Errorf("unable to read resources from type: %s", m.Kind)
		}
		return r, nil

	}

	// The default behavior is to just return the node itself
	return kio.ResourceNodeSlice{node}, nil
}

// clean is used to discover readers which implement `cleaner` and invoke their `Clean` function.
func clean() (cleanOpt Option, doClean func()) {
	// The cleaner interface can be implemented by readers to implement clean up logic after a filter iteration
	type cleaner interface{ Clean() error }
	var cleaners []cleaner

	// Accumulate cleaner instances using a reader option
	cleanOpt = func(r kio.Reader) kio.Reader {
		if c, ok := r.(cleaner); ok {
			cleaners = append(cleaners, c)
		}
		return r
	}

	// Invoke all the `cleaner.Clean()` functions
	doClean = func() {
		for _, c := range cleaners {
			// TODO This should produce warnings, maybe the errors can be accumulated on the filer itself
			_ = c.Clean()
		}
	}

	return
}
