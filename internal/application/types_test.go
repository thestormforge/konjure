/*
Copyright 2022 CloudBolt, Inc.

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

package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupKind_String(t *testing.T) {
	cases := []struct {
		desc      string
		groupKind GroupKind
		expected  string
	}{
		{
			desc:     "empty",
			expected: ".",
		},
		{
			// This case is important because of how `kubectl` resolves types:
			// for example, `kubectl get Foo` won't work (it's a plain kind, not
			// a resource name); but `kubectl get Foo.` will trigger a GVK parse
			// that will ultimately resolve to the correct type.
			desc:      "kind only",
			groupKind: GroupKind{Kind: "Foo"},
			expected:  "Foo.",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.groupKind.String())
		})
	}
}
