/*
Copyright 2022 GramLabs, Inc.

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

package pipes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thestormforge/konjure/pkg/filters"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestHelmValues_Read(t *testing.T) {
	cases := []struct {
		desc     string
		reader   HelmValues
		expected []*yaml.RNode
	}{
		{
			desc: "single flat set",
			reader: HelmValues{
				Values: []string{
					"foo=bar",
				},
			},
			expected: []*yaml.RNode{yaml.MustParse(`foo: bar`)},
		},
		{
			desc: "multiple flat set",
			reader: HelmValues{
				Values: []string{
					"a=b",
					"c=d",
				},
			},
			expected: []*yaml.RNode{
				yaml.MustParse(`
a: b
c: d`),
			},
		},
		{
			desc: "single nested set",
			reader: HelmValues{
				Values: []string{
					"foo[0]=bar",
				},
			},
			expected: []*yaml.RNode{yaml.MustParse(`foo:
- bar`)},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			actual, err := tc.reader.Read()
			if assert.NoError(t, err) {
				actualString, err := kio.StringAll(actual)
				require.NoError(t, err, "failed to string node")
				expectedString, err := kio.StringAll(tc.expected)
				require.NoError(t, err, "failed to string node")
				assert.YAMLEq(t, expectedString, actualString)
			}
		})
	}
}

func TestHelmValues_Apply(t *testing.T) {
	nodes, err := (&filters.Pipeline{
		Inputs: []kio.Reader{
			&HelmValues{Values: []string{"a=b"}},
		},
		Filters: []kio.Filter{
			kio.FilterAll((&HelmValues{Values: []string{
				"c.d=e",
				"a=z",
			}}).Apply()),
		},
	}).Read()
	assert.NoError(t, err, "failed to apply")

	actual, err := nodes[0].MarshalJSON()
	assert.NoError(t, err, "failed to produce JSON")
	assert.JSONEq(t, `{"a":"z","c":{"d":"e"}}`, string(actual))
}

func TestHelmValues_Flatten(t *testing.T) {
	flattened, err := (&filters.Pipeline{
		Inputs: []kio.Reader{
			&HelmValues{Values: []string{"a=b"}},
			&HelmValues{Values: []string{"c=d"}},
		},
		Filters: []kio.Filter{(&HelmValues{}).Flatten()},
	}).Read()
	require.NoError(t, err, "failed to flatten output")
	assert.Len(t, flattened, 1)

	out := struct {
		A string `yaml:"a"`
		C string `yaml:"c"`
	}{}
	err = flattened[0].YNode().Decode(&out)
	require.NoError(t, err, "failed to decode result")
	assert.Equal(t, "b", out.A)
	assert.Equal(t, "d", out.C)
}
