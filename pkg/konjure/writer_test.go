package konjure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestRestoreWhiteSpace(t *testing.T) {
	cases := []struct {
		desc           string
		roundTrippable bool
		input          string
	}{
		{
			desc:           "multiple blank lines",
			roundTrippable: true,
			input: `test:
- foo



test2:
- bar
`,
		},
		{
			desc:           "single line head comment",
			roundTrippable: true,
			input: `a: b

# foobar
c: d
`,
		},
		{
			desc:           "multi line head comment",
			roundTrippable: true,
			input: `a: b

# foo
# bar
c: d
`,
		},
		{
			desc:           "multi line head comment and multiple blank lines",
			roundTrippable: true,
			input: `a: b


# foo
# bar
c: d
`,
		},
		{
			desc:           "single line foot comment",
			roundTrippable: true,
			input: `a: b
# foo

c: d
`,
		},
		{
			desc:           "multi line foot comment",
			roundTrippable: true,
			input: `a: b
# foo
# bar

c: d
`,
		},
		{
			desc:           "multi line foot comment and multiple blank lines",
			roundTrippable: true,
			input: `a: b
# foo
# bar


c: d
`,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			node, err := yaml.Parse(c.input)
			require.NoError(t, err, "invalid test input YAML")
			restoreVerticalWhiteSpace([]*yaml.RNode{node})
			actual, err := kio.StringAll([]*yaml.RNode{node})
			require.NoError(t, err, "failed to format YAML")
			if c.roundTrippable {
				assert.Equal(t, c.input, actual)
			} else {
				assert.NotEqual(t, c.input, actual)
			}
		})
	}
}

func TestSplitColumns(t *testing.T) {
	cases := []struct {
		desc            string
		spec            string
		expectedHeaders []string
		expectedColumns []string
	}{
		{
			desc:            "default headers",
			spec:            "foo, bar",
			expectedHeaders: []string{"FOO", "BAR"},
			expectedColumns: []string{"foo", "bar"},
		},
		{
			desc:            "default headers from path",
			spec:            "a.b.c.foo, x.y.z.bar",
			expectedHeaders: []string{"FOO", "BAR"},
			expectedColumns: []string{"a.b.c.foo", "x.y.z.bar"},
		},
		{
			desc:            "explicit headers",
			spec:            "Foo:a.b.c.foo  ,  Bar:x.y.z.bar",
			expectedHeaders: []string{"Foo", "Bar"},
			expectedColumns: []string{"a.b.c.foo", "x.y.z.bar"},
		},
		{
			desc:            "escaped column",
			spec:            ":x:y:z, :a:b.c",
			expectedHeaders: []string{"X:Y:Z", "C"},
			expectedColumns: []string{"x:y:z", "a:b.c"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			headers, columns := splitColumns(tc.spec)
			assert.Equal(t, tc.expectedHeaders, headers)
			assert.Equal(t, tc.expectedColumns, columns)
		})
	}
}
