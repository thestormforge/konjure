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
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSelectorFilter_Filter(t *testing.T) {
	cases := []struct {
		desc     string
		selector SelectorFilter
		input    []*yaml.RNode
		expected []*yaml.RNode
	}{
		{
			desc: "match all",
			input: []*yaml.RNode{
				rmNode("test", nil, nil),
			},
			expected: []*yaml.RNode{
				rmNode("test", nil, nil),
			},
		},
		{
			desc: "match annotation",
			selector: SelectorFilter{
				AnnotationSelector: "test=testing",
			},
			input: []*yaml.RNode{
				rmNode("test", nil, nil),
				rmNode("testWithAnnotation", nil, map[string]string{"test": "testing"}),
			},
			expected: []*yaml.RNode{
				rmNode("testWithAnnotation", nil, map[string]string{"test": "testing"}),
			},
		},
		{
			desc: "match annotation negate",
			selector: SelectorFilter{
				AnnotationSelector: "test=testing",
				Negate:             true,
			},
			input: []*yaml.RNode{
				rmNode("test", nil, nil),
				rmNode("testWithAnnotation", nil, map[string]string{"test": "testing"}),
			},
			expected: []*yaml.RNode{
				rmNode("test", nil, nil),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			actual, err := c.selector.Filter(c.input)
			if assert.NoError(t, err) {
				assert.Equal(t, c.expected, actual)
			}
		})
	}
}

// node returns an RNode representing the supplied resource metadata.
func rmNode(name string, labels, annotations map[string]string) *yaml.RNode {
	data, err := yaml.Marshal(&yaml.ResourceMeta{
		TypeMeta: yaml.TypeMeta{APIVersion: "invalid.example.com/v1", Kind: "Test"},
		ObjectMeta: yaml.ObjectMeta{
			NameMeta: yaml.NameMeta{
				Name: name,
			},
			Labels:      labels,
			Annotations: annotations,
		},
	})
	if err != nil {
		panic(err)
	}
	return yaml.MustParse(string(data))
}
