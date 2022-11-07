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
