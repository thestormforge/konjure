package filters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldPath(t *testing.T) {
	cases := []struct {
		desc     string
		path     string
		data     map[string]string
		expected []string
	}{
		{
			desc: "empty",
		},
		{
			desc:     "leading slash",
			path:     "/foo/bar",
			expected: []string{"foo", "bar"},
		},
		{
			desc:     "leading slashes",
			path:     "////foo/bar",
			expected: []string{"foo", "bar"},
		},
		{
			desc:     "template",
			path:     "/foo/[bar={.x}]",
			data:     map[string]string{"x": "test"},
			expected: []string{"foo", "[bar=test]"},
		},
		{
			desc:     "nested slash",
			path:     "/foo/[bar=a/b]",
			expected: []string{"foo", "[bar=a/b]"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			actual, err := FieldPath(tc.path, tc.data)
			if assert.NoError(t, err) {
				assert.Equal(t, tc.expected, actual)
			}
		})
	}
}
