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
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestResource_GetRNode(t *testing.T) {
	cases := []struct {
		desc     string
		resource Resource
		expected *yaml.RNode
	}{
		{
			desc: "helm",
			resource: Resource{
				Helm: &konjurev1beta2.Helm{Chart: "test"},
			},
			expected: mustRNode(&konjurev1beta2.Helm{Chart: "test"}),
		},
		{
			desc: "git",
			resource: Resource{
				Git: &konjurev1beta2.Git{Repository: url.URL{Scheme: "http", Host: "example.com", Path: "/repo"}},
			},
			expected: mustRNode(&konjurev1beta2.Git{Repository: url.URL{Scheme: "http", Host: "example.com", Path: "/repo"}}),
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			actual, err := c.resource.GetRNode()
			if assert.NoError(t, err) {
				assert.Equal(t, c.expected, actual)
			}
		})
	}
}

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
					Name: "/this/is/a/test",
				},
				str: "/this/is/a/test",
			},
		},
		{
			desc:    "file object",
			rawJSON: `{"file":{"name":"/this/is/a/test"}}`,
			expected: Resource{
				File: &konjurev1beta2.File{
					Name: "/this/is/a/test",
				},
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

func mustRNode(obj interface{}) *yaml.RNode {
	rn, err := konjurev1beta2.GetRNode(obj)
	if err != nil {
		panic(err)
	}
	return rn
}
