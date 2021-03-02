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
	"io"

	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// New returns a resource node reader or nil if the input is not recognized.
func New(obj interface{}) kio.Reader {
	switch r := obj.(type) {
	case *konjurev1beta2.Resource:
		return &ResourceReader{Resources: r.Resources}
	case *konjurev1beta2.Helm:
		return NewHelmReader(r)
	case *konjurev1beta2.Jsonnet:
		return NewJsonnetReader(r)
	case *konjurev1beta2.Kubernetes:
		return NewKubernetesReader(r)
	case *konjurev1beta2.Kustomize:
		return NewKustomizeReader(r)
	case *konjurev1beta2.Secret:
		return &SecretReader{Secret: *r}
	case *konjurev1beta2.Git:
		return &GitReader{Git: *r}
	case *konjurev1beta2.HTTP:
		return &HTTPReader{HTTP: *r}
	case *konjurev1beta2.File:
		return &FileReader{File: *r}
	default:
		return nil
	}
}

// Filter is a KYAML Filter that maps Konjure resource specifications to
// KYAML Readers, then reads and flattens the resulting RNodes into the final
// result. Due to the recursive nature of this filter, the depth (number of
// allowed recursive iterations) must be specified; the default value of 0 is
// effectively a no-op.
type Filter struct {
	// The number of iterations to perform when expanding Kojure resources.
	Depth int
	// The reader to use for an empty specification, defaults to stdin.
	DefaultReader io.Reader
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

	var cleaners Cleaners
	defer func() {
		// TODO This should produce warnings, maybe the errors can be accumulated on the filer itself
		cleaners.CleanUp()
	}()

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

		// Create a reader
		r := New(obj)
		if r == nil {
			return nil, fmt.Errorf("unable to read resources from type: %s", m.Kind)
		}

		// If the reader needs a default stream, set it
		if rr, ok := r.(*ResourceReader); ok && rr.Reader == nil {
			rr.Reader = f.DefaultReader
		}

		// If a reader requires clean up, add it to the list
		if c, ok := r.(Cleaner); ok {
			cleaners = append(cleaners, c)
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
