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

package konjure

import (
	"io"

	"github.com/thestormforge/konjure/internal/filters"
	"github.com/thestormforge/konjure/internal/readers"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kiofilters "sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Filter replaces Konjure resources with the expanded resources they represent.
type Filter struct {
	// The number of times to recursively filter the resource list.
	Depth int
	// The default reader to use, defaults to stdin.
	DefaultReader io.Reader
	// Label selector of resources to retain.
	LabelSelector string
	// Annotation selector of resources to retain.
	AnnotationSelector string
	// Flag indicating that status fields should not be stripped.
	KeepStatus bool
	// Flag indicating that comments should not be stripped.
	KeepComments bool
	// Flag indicating that output should be formatted.
	Format bool
}

// Filter expands all of the Konjure resources using the configured executors.
func (f *Filter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	var err error

	nodes, err = (&readers.ReadersFilter{Depth: f.Depth, DefaultReader: f.DefaultReader}).Filter(nodes)
	if err != nil {
		return nil, err
	}

	nodes, err = (&filters.SelectorFilter{LabelSelector: f.LabelSelector, AnnotationSelector: f.AnnotationSelector}).Filter(nodes)
	if err != nil {
		return nil, err
	}

	if !f.KeepStatus {
		nodes, err = kio.FilterAll(yaml.Clear("status")).Filter(nodes)
		if err != nil {
			return nil, err
		}
	}

	if !f.KeepComments {
		nodes, err = (&kiofilters.StripCommentsFilter{}).Filter(nodes)
		if err != nil {
			return nil, err
		}
	}

	if f.Format {
		nodes, err = (&kiofilters.FormatFilter{}).Filter(nodes)
		if err != nil {
			return nil, err
		}
	}

	return nodes, nil
}
