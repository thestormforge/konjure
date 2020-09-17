/*
Copyright 2020 GramLabs, Inc.

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

package version

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewResource(t *testing.T) {
	cases := []struct {
		desc     string
		target   string
		expected Resource
	}{
		// NOTE: The test will set `expected.Target = target` to reduce duplication

		{
			desc: "empty",
		},

		// These are the examples from the Kustomize API Reference for specifying resources

		{
			desc:   "kustomize api reference 1",
			target: "myNamespace.yaml",
		},
		{
			desc:   "kustomize api reference 2",
			target: "sub-dir/some-deployment.yaml",
		},
		{
			desc:   "kustomize api reference 3",
			target: "../../commonbase",
		},
		{
			desc:   "kustomize api reference 4",
			target: "github.com/kubernetes-sigs/kustomize/examples/multibases?ref=v1.0.6",
			expected: Resource{
				Version:    "v1.0.6",
				ImageNames: []string{"kubernetes-sigs/kustomize"},
				ImageTag:   "1.0.6",
			},
		},
		{
			desc:   "kustomize api reference 5",
			target: "deployment.yaml",
		},
		{
			desc:   "kustomize api reference 6",
			target: "github.com/kubernets-sigs/kustomize/examples/helloWorld?ref=test-branch",
			expected: Resource{
				ImageNames: []string{"kubernets-sigs/kustomize"},
				ImageTag:   "test-branch",
			},
		},

		// More random examples

		{
			desc:   "arbitrary HTTP",
			target: "http://invalid.example.com/testing",
		},
		{
			desc:   "no reference",
			target: "git::https://github.com/example/example",
			expected: Resource{
				ImageNames: []string{"example/example"},
				ImageTag:   "edge",
			},
		},
		{
			desc:   "edge reference",
			target: "git::https://github.com/example/example?ref=master",
			expected: Resource{
				ImageNames: []string{"example/example"},
				ImageTag:   "edge",
			},
		},
		{
			desc:   "repository root",
			target: "git::https://github.com/example/example?ref=v1.0.0",
			expected: Resource{
				Version:    "v1.0.0",
				ImageNames: []string{"example/example"},
				ImageTag:   "1.0.0",
			},
		},
		{
			desc:   "repository base subdirectory",
			target: "git::https://github.com/example/example/base?ref=v1.0.0",
			expected: Resource{
				Version:    "v1.0.0",
				ImageNames: []string{"example/example"},
				ImageTag:   "1.0.0",
			},
		},
		{
			desc:   "repository k8s subdirectory",
			target: "git::https://github.com/example/example/k8s/base?ref=v1.0.0",
			expected: Resource{
				Version:    "v1.0.0",
				ImageNames: []string{"example/example"},
				ImageTag:   "1.0.0",
			},
		},
		{
			desc:   "repository mixed case", // Seriously? Who does that?
			target: "git::https://github.com/Example/Example?ref=v1.0.0",
			expected: Resource{
				Version:    "v1.0.0",
				ImageNames: []string{"example/example"},
				ImageTag:   "1.0.0",
			},
		},
		{
			desc:   "non-GitHub",
			target: "git::file:///var/tmp/git/example?ref=v1.0.0",
			expected: Resource{
				Version:  "v1.0.0",
				ImageTag: "1.0.0",
			},
		},
		{
			desc:   "hash reference",
			target: "git::file:///var/tmp/git/example?ref=69ead85207e924b118aab0f51506ef76b187b734",
			expected: Resource{
				ImageTag: "sha-69ead852",
			},
		},
	}
	for _, c := range cases {
		// This just prevents massive duplication of the target
		c.expected.Target = c.target

		t.Run(c.desc, func(t *testing.T) {
			assert.Equal(t, &c.expected, NewResource(c.target))
		})
	}
}

func TestResource_MatchedImageName(t *testing.T) {
	cases := []struct {
		desc     string
		resource Resource
		image    string

		expected string
	}{
		{
			desc:     "without tag",
			resource: Resource{ImageNames: []string{"foo/bar"}},
			image:    "foo/bar",
			expected: "foo/bar",
		},
		{
			desc:     "with tag",
			resource: Resource{ImageNames: []string{"foo/bar"}},
			image:    "foo/bar:test",
			expected: "foo/bar",
		},
		{
			desc:     "with digest",
			resource: Resource{ImageNames: []string{"foo/bar"}},
			image:    "foo/bar@sha256:77af4d6b9913e693e8d0b4b294fa62ade6054e6b2f1ffb617ac955dd63fb0182",
			expected: "foo/bar",
		},
		{
			desc:     "with registry port number",
			resource: Resource{ImageNames: []string{"registry.example:1234/foo/bar"}},
			image:    "registry.example:1234/foo/bar",
			expected: "registry.example:1234/foo/bar",
		},
		{
			desc:     "with registry port number and tag",
			resource: Resource{ImageNames: []string{"registry.example:1234/foo/bar"}},
			image:    "registry.example:1234/foo/bar:test",
			expected: "registry.example:1234/foo/bar",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert.Equal(t, c.expected, c.resource.MatchedImageName(c.image))
		})
	}
}

func TestResource_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		desc     string
		input    string
		expected Resource
	}{
		{
			desc:  "string",
			input: `"foobar"`,
			expected: Resource{
				Target: "foobar",
			},
		},
		{
			desc:  "object",
			input: `{"target":"foobar"}`,
			expected: Resource{
				Target: "foobar",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			actual := Resource{}
			err := json.Unmarshal([]byte(c.input), &actual)
			if assert.NoError(t, err) {
				assert.Equal(t, c.expected, actual)
			}
		})
	}
}
