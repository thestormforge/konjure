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
	"strings"

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

var _ kio.Filter = &Filter{}

// Filter expands all of the Konjure resources using the configured executors.
func (f *Filter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	return f.filterToDepth(nodes, f.Depth)
}

// filterToDepth applies the expansion executors up to the specified depth (i.e. a File executor that produces a
// Kustomize resource would be at a depth of 2).
func (f *Filter) filterToDepth(nodes []*yaml.RNode, depth int) ([]*yaml.RNode, error) {
	if depth <= 0 {
		return nodes, nil
	}

	// Create a new cleaner for this iteration of expansion
	var cleaners Cleaners
	defer func() {
		// TODO This should produce warnings, maybe the errors can be accumulated on the filer itself
		cleaners.CleanUp()
	}()
	opts := append([]Option{cleaners.Register}, f.ReaderOptions...)

	var result []*yaml.RNode
	var depthNext int
	for _, n := range nodes {
		m, err := n.GetMeta()
		if err != nil {
			return nil, err
		}

		// Just include non-Konjure resources directly
		if m.APIVersion != konjurev1beta2.APIVersion {
			result = append(result, n)
			continue
		}

		// Only set the depth if we encounter a Konjure resource (which could expand into other Konjure resources)
		depthNext = depth - 1

		// Create a new typed object from the YAML
		obj, err := konjurev1beta2.NewForType(&m.TypeMeta)
		if err != nil {
			return nil, err
		}
		if err := n.YNode().Decode(obj); err != nil {
			return nil, err
		}

		// Create a new reader
		r := New(obj, opts...)
		if r == nil {
			return nil, fmt.Errorf("unable to read resources from type: %s", m.Kind)
		}

		// Accumulate additional resource nodes
		ns, err := r.Read()
		if err != nil {
			return nil, err
		}
		result = append(result, ns...)
	}

	return f.filterToDepth(result, depthNext)
}

// CleanUpError is an aggregation of errors that occur during clean up.
type CleanUpError []error

// Error returns the newline delimited error strings of all the aggregated errors.
func (e CleanUpError) Error() string {
	var errStrings []string
	for _, err := range e {
		errStrings = append(errStrings, err.Error())
	}
	return strings.Join(errStrings, "\n")
}

// Cleaner is an interface for readers that may need to perform clean up of temporary resources.
type Cleaner interface {
	Clean() error
}

// Cleaners is a collection of cleaners that can be invoked together.
type Cleaners []Cleaner

// Register the supplied reader with this cleaner.
func (cs *Cleaners) Register(r kio.Reader) kio.Reader {
	if c, ok := r.(Cleaner); ok {
		*cs = append(*cs, c)
	}
	return r
}

// CleanUp invokes all of the cleaners, individual failures are aggregated and
// will not prevent other clean up tasks from being executed.
func (cs Cleaners) CleanUp() error {
	var errs CleanUpError
	for _, c := range cs {
		if err := c.Clean(); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}
