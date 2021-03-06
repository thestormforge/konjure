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

package spec

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func TestParser_Decode(t *testing.T) {
	cases := []struct {
		desc     string
		parser   Parser
		spec     string
		expected interface{}
	}{
		{
			desc:     "default reader",
			spec:     "-",
			parser:   Parser{Reader: strings.NewReader("test")},
			expected: &kio.ByteReader{Reader: strings.NewReader("test")},
		},
		{
			desc: "postgres example",
			spec: "github.com/thestormforge/examples/postgres/application",
			expected: &konjurev1beta2.Git{
				Repository: "https://github.com/thestormforge/examples.git",
				Context:    "postgres/application",
			},
		},
		{
			desc: "kubernetes default deployments of application 'test'",
			spec: "k8s:default/deployments?labelSelector=app.kubernetes.io/name%3Dtest",
			expected: &konjurev1beta2.Kubernetes{
				Namespaces:    []string{"default"},
				Types:         []string{"deployments"},
				LabelSelector: "app.kubernetes.io/name=test",
			},
		},
		{
			desc: "kubernetes default namespace",
			spec: "k8s:default",
			expected: &konjurev1beta2.Kubernetes{
				Namespaces: []string{"default"},
			},
		},
		{
			desc: "kubernetes all deployments",
			spec: "k8s:/deployments",
			expected: &konjurev1beta2.Kubernetes{
				Namespaces: nil,
				Types:      []string{"deployments"},
			},
		},

		// These are a bunch of test cases from Kustomize for Git URLs
		{
			desc: "kustomize-tc-0",
			spec: "https://git-codecommit.us-east-2.amazonaws.com/someorg/somerepo/somedir",
			expected: &konjurev1beta2.Git{
				Repository: "https://git-codecommit.us-east-2.amazonaws.com/someorg/somerepo",
				Context:    "somedir",
			},
		},
		{
			desc: "kustomize-tc-1",
			spec: "https://git-codecommit.us-east-2.amazonaws.com/someorg/somerepo/somedir?ref=testbranch",
			expected: &konjurev1beta2.Git{
				Repository: "https://git-codecommit.us-east-2.amazonaws.com/someorg/somerepo",
				Context:    "somedir",
				Refspec:    "testbranch",
			},
		},
		{
			desc: "kustomize-tc-2",
			spec: "https://fabrikops2.visualstudio.com/someorg/somerepo?ref=master",
			expected: &konjurev1beta2.Git{
				Repository: "https://fabrikops2.visualstudio.com/someorg/somerepo",
				Refspec:    "master",
			},
		},
		{
			desc: "kustomize-tc-3",
			spec: "http://github.com/someorg/somerepo/somedir",
			expected: &konjurev1beta2.Git{
				Repository: "https://github.com/someorg/somerepo.git",
				Context:    "somedir",
			},
		},
		{
			desc: "kustomize-tc-4",
			spec: "git@github.com:someorg/somerepo/somedir",
			expected: &konjurev1beta2.Git{
				Repository: "git@github.com:someorg/somerepo.git",
				Context:    "somedir",
			},
		},
		// This doesn't seem valid, an SCP-like spec can't have a port number
		//{
		//	desc: "kustomize-tc-5",
		//	spec: "git@gitlab2.sqtools.ru:10022/infra/kubernetes/thanos-base.git?ref=v0.1.0",
		//	expected: &konjurev1beta2.Git{
		//		Repository: url.URL{User: url.User("git"), Host: "gitlab2.sqtools.ru:10022", Path: "/infra/kubernetes/thanos-base.git"},
		//		Refspec:    "v0.1.0",
		//	},
		//},
		{
			desc: "kustomize-tc-6",
			spec: "git@bitbucket.org:company/project.git//path?ref=branch",
			expected: &konjurev1beta2.Git{
				Repository: "git@bitbucket.org:company/project.git",
				Context:    "path",
				Refspec:    "branch",
			},
		},
		{
			desc: "kustomize-tc-7",
			spec: "https://itfs.mycompany.com/collection/project/_git/somerepos",
			expected: &konjurev1beta2.Git{
				Repository: "https://itfs.mycompany.com/collection/project/_git/somerepos",
			},
		},
		{
			desc: "kustomize-tc-8",
			spec: "https://itfs.mycompany.com/collection/project/_git/somerepos?version=v1.0.0",
			expected: &konjurev1beta2.Git{
				Repository: "https://itfs.mycompany.com/collection/project/_git/somerepos",
				Refspec:    "v1.0.0",
			},
		},
		{
			desc: "kustomize-tc-9",
			spec: "https://itfs.mycompany.com/collection/project/_git/somerepos/somedir?version=v1.0.0",
			expected: &konjurev1beta2.Git{
				Repository: "https://itfs.mycompany.com/collection/project/_git/somerepos",
				Context:    "somedir",
				Refspec:    "v1.0.0",
			},
		},
		{
			desc: "kustomize-tc-10",
			spec: "git::https://itfs.mycompany.com/collection/project/_git/somerepos",
			expected: &konjurev1beta2.Git{
				Repository: "https://itfs.mycompany.com/collection/project/_git/somerepos",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			actual, err := c.parser.Decode(c.spec)
			if assert.NoError(t, err) {
				assert.Equal(t, c.expected, actual)
			}
		})
	}
}

func TestParseSpecFailures(t *testing.T) {
	cases := []struct {
		desc      string
		reader    Parser
		spec      string
		errString string
	}{
		//{},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			_, err := c.reader.Decode(c.spec)
			assert.EqualError(t, err, c.errString)
		})
	}
}
