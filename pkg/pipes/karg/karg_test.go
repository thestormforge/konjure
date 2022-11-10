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

package karg

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithGetOptions(t *testing.T) {
	cases := []struct {
		desc     string
		args     []string
		opts     []GetOption
		expected []string
	}{
		{
			desc:     "empty",
			expected: []string{"kubectl", "get"},
		},
		{
			desc:     "resource kind with selector",
			opts:     []GetOption{ResourceKind("v1", "Namespace"), Selector("foo=bar")},
			expected: []string{"kubectl", "get", "Namespace.v1.", "--selector", "foo=bar"},
		},
		{
			desc:     "resource types with all namespaces",
			args:     []string{"--namespace", "default"},
			opts:     []GetOption{ResourceType("deployments", "statefulsets"), AllNamespaces(true)},
			expected: []string{"kubectl", "get", "deployments,statefulsets", "--all-namespaces"},
		},
		{
			desc:     "tricky namespace",
			args:     []string{"--namespace=default"},
			opts:     []GetOption{ResourceName("secret", "my-token"), AllNamespaces(true)},
			expected: []string{"kubectl", "get", "secret/my-token", "--all-namespaces"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			cmd := exec.Command("kubectl", append([]string{"get"}, tc.args...)...)
			WithGetOptions(cmd, tc.opts...)
			assert.Equal(t, tc.expected, cmd.Args)
		})
	}
}
