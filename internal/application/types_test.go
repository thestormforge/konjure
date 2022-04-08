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
