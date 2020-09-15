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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestExtractInfo(t *testing.T) {
	cases := []struct {
		desc     string
		origin   string
		expected *OriginInfo
	}{
		{
			desc: "empty",
		},

		// These are the examples from the Kustomize API Reference for specifying resources

		{
			desc:   "kustomize api reference 1",
			origin: "myNamespace.yaml",
		},
		{
			desc:   "kustomize api reference 2",
			origin: "sub-dir/some-deployment.yaml",
		},
		{
			desc:   "kustomize api reference 3",
			origin: "../../commonbase",
		},
		{
			desc:   "kustomize api reference 4",
			origin: "github.com/kubernetes-sigs/kustomize/examples/multibases?ref=v1.0.6",
			expected: &OriginInfo{
				Version:    "v1.0.6",
				ImageNames: []string{"kubernetes-sigs/kustomize"},
				ImageTag:   "1.0.6",
			},
		},
		{
			desc:   "kustomize api reference 5",
			origin: "deployment.yaml",
		},
		{
			desc:   "kustomize api reference 6",
			origin: "github.com/kubernets-sigs/kustomize/examples/helloWorld?ref=test-branch",
			expected: &OriginInfo{
				Version:    "",
				ImageNames: []string{"kubernets-sigs/kustomize"},
				ImageTag:   "test-branch",
			},
		},

		// More random examples

		{
			desc:   "arbitrary HTTP",
			origin: "http://invalid.example.com/testing",
		},
		{
			desc:   "no reference",
			origin: "git::https://github.com/example/example",
			expected: &OriginInfo{
				Version:    "",
				ImageNames: []string{"example/example"},
				ImageTag:   "edge",
			},
		},
		{
			desc:   "edge reference",
			origin: "git::https://github.com/example/example?ref=master",
			expected: &OriginInfo{
				Version:    "",
				ImageNames: []string{"example/example"},
				ImageTag:   "edge",
			},
		},
		{
			desc:   "repository root",
			origin: "git::https://github.com/example/example?ref=v1.0.0",
			expected: &OriginInfo{
				Version:    "v1.0.0",
				ImageNames: []string{"example/example"},
				ImageTag:   "1.0.0",
			},
		},
		{
			desc:   "repository base subdirectory",
			origin: "git::https://github.com/example/example/base?ref=v1.0.0",
			expected: &OriginInfo{
				Version:    "v1.0.0",
				ImageNames: []string{"example/example"},
				ImageTag:   "1.0.0",
			},
		},
		{
			desc:   "repository k8s subdirectory",
			origin: "git::https://github.com/example/example/k8s/base?ref=v1.0.0",
			expected: &OriginInfo{
				Version:    "v1.0.0",
				ImageNames: []string{"example/example"},
				ImageTag:   "1.0.0",
			},
		},
		{
			desc:   "repository mixed case", // Seriously? Who does that?
			origin: "git::https://github.com/Example/Example?ref=v1.0.0",
			expected: &OriginInfo{
				Version:    "v1.0.0",
				ImageNames: []string{"example/example"},
				ImageTag:   "1.0.0",
			},
		},
		{
			desc:   "non-GitHub",
			origin: "git::file:///var/tmp/git/example?ref=v1.0.0",
			expected: &OriginInfo{
				Version:  "v1.0.0",
				ImageTag: "1.0.0",
			},
		},
		{
			desc:   "hash reference",
			origin: "git::file:///var/tmp/git/example?ref=69ead85207e924b118aab0f51506ef76b187b734",
			expected: &OriginInfo{
				Version:  "",
				ImageTag: "sha-69ead852",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert.Equal(t, c.expected, ExtractInfo(c.origin))
		})
	}
}

func TestOriginInfo_MatchesImage(t *testing.T) {
	cases := []struct {
		desc     string
		image    string
		info     OriginInfo
		expected bool
	}{
		{
			desc:     "without tag",
			image:    "foo/bar",
			info:     OriginInfo{ImageNames: []string{"foo/bar"}},
			expected: true,
		},
		{
			desc:     "with tag",
			image:    "foo/bar:test",
			info:     OriginInfo{ImageNames: []string{"foo/bar"}},
			expected: true,
		},
		{
			desc:     "with digest",
			image:    "foo/bar@sha256:77af4d6b9913e693e8d0b4b294fa62ade6054e6b2f1ffb617ac955dd63fb0182",
			info:     OriginInfo{ImageNames: []string{"foo/bar"}},
			expected: true,
		},
		{
			desc:     "with registry port number",
			image:    "registry.example:1234/foo/bar",
			info:     OriginInfo{ImageNames: []string{"registry.example:1234/foo/bar"}},
			expected: true,
		},
		{
			desc:     "with registry port number and tag",
			image:    "registry.example:1234/foo/bar:test",
			info:     OriginInfo{ImageNames: []string{"registry.example:1234/foo/bar"}},
			expected: true,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			_, ok := c.info.MatchesImage(c.image)
			assert.Equal(t, c.expected, ok)
		})
	}
}

func TestOriginInfo_Filter(t *testing.T) {
	cases := []struct {
		desc     string
		info     OriginInfo
		image    string
		expected string
	}{
		{
			desc:     "matches",
			info:     OriginInfo{ImageNames: []string{"foo/bar"}, ImageTag: "1.0.0"},
			image:    "foo/bar",
			expected: "foo/bar:1.0.0",
		},
		{
			desc:     "no match",
			info:     OriginInfo{ImageNames: []string{"foo/bar"}, ImageTag: "1.0.0"},
			image:    "foo/test",
			expected: "foo/test",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			n, err := c.info.Filter(yaml.NewScalarRNode(c.image))
			require.NoError(t, err)
			assert.Equal(t, c.expected, n.YNode().Value)
		})
	}
}
