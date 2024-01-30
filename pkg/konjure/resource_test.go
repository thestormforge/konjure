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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestResource_Read(t *testing.T) {
	cases := []struct {
		desc     string
		resource Resource
		expected []*yaml.RNode
	}{
		{
			desc: "helm",
			resource: Resource{
				Helm: &konjurev1beta2.Helm{Chart: "test"},
			},
			expected: []*yaml.RNode{mustRNode(&konjurev1beta2.Helm{Chart: "test"})},
		},
		{
			desc: "git",
			resource: Resource{
				Git: &konjurev1beta2.Git{Repository: "http://example.com/repo"},
			},
			expected: []*yaml.RNode{mustRNode(&konjurev1beta2.Git{Repository: "http://example.com/repo"})},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			actual, err := c.resource.Read()
			if assert.NoError(t, err) {
				assert.Equal(t, c.expected, actual)
			}
		})
	}
}

const testResource = `apiVersion: invalid.example.com/v1
kind: Test
metadata:
  name: this-is-a-test
`

func TestResource_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		rawJSON  string
		expected Resource
	}{
		{
			desc:    "file string",
			rawJSON: `"/this/is/a/test"`,
			expected: Resource{
				File: &konjurev1beta2.File{
					Path: "/this/is/a/test",
				},
				str: "/this/is/a/test",
			},
		},
		{
			desc:    "file object",
			rawJSON: `{"file":{"path":"/this/is/a/test"}}`,
			expected: Resource{
				File: &konjurev1beta2.File{
					Path: "/this/is/a/test",
				},
			},
		},
		{
			desc:    "data",
			rawJSON: `"data:;base64,` + base64.URLEncoding.EncodeToString([]byte(testResource)) + `"`,
			expected: Resource{
				raw: &kio.ByteReader{Reader: bytes.NewReader([]byte(testResource))},
				str: `data:;base64,` + base64.URLEncoding.EncodeToString([]byte(testResource)),
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			actual := Resource{}
			err := json.Unmarshal([]byte(c.rawJSON), &actual)
			if assert.NoError(t, err) {
				assert.Equal(t, c.expected, actual)
			}
		})
	}
}

func TestResource_MarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		resource Resource
		expected string
	}{
		{
			desc:     "file string",
			resource: Resource{str: "/this/is/a/test", File: &konjurev1beta2.File{Path: "/this/is/a/test"}},
			expected: `"/this/is/a/test"`,
		},
		{
			desc:     "file object",
			resource: Resource{File: &konjurev1beta2.File{Path: "/this/is/a/test"}},
			expected: `"/this/is/a/test"`,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			data, err := json.Marshal(&c.resource)
			if assert.NoError(t, err) {
				assert.JSONEq(t, c.expected, string(data))
			}
		})
	}
}

func TestResource_DeepCopyInto(t *testing.T) {
	// Quick sanity test to make sure we keep calm and don't panic
	in := Resource{str: "test", File: &konjurev1beta2.File{Path: "test"}}
	out := Resource{}
	in.DeepCopyInto(&out)
	assert.Equal(t, in.str, out.str)
	assert.Equal(t, in.File, out.File)
	assert.NotSame(t, in.File, out.File)
}

func mustRNode(obj any) *yaml.RNode {
	rn, err := konjurev1beta2.GetRNode(obj)
	if err != nil {
		panic(err)
	}
	return rn
}
