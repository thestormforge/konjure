package filters

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
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

func TestSetValues(t *testing.T) {
	cases := []struct {
		desc        string
		inputs      []*yaml.RNode
		specs       []string
		forceString bool
		expected    string
	}{
		{
			desc: "empty",
		},
		{
			desc:   "simple name",
			inputs: []*yaml.RNode{yaml.MustParse(`foo: bar`)},
			specs:  []string{"abc=xyz"},
			expected: `#
foo: bar
abc: xyz
`,
		},
		{
			desc:   "indexed name",
			inputs: []*yaml.RNode{yaml.MustParse(`foo: bar`)},
			specs:  []string{"a.b.[c=d].e=wxyz"},
			expected: `#
foo: bar
a:
  b:
  - c: d
    e: wxyz
`,
		},
		{
			desc: "no create null",
			inputs: []*yaml.RNode{yaml.MustParse(`#
foo: bar
`)},
			specs: []string{"abc=null"},
			expected: `#
foo: bar
`,
		},
		{
			desc: "unset",
			inputs: []*yaml.RNode{yaml.MustParse(`#
foo: bar
abc: xyz
`)},
			specs: []string{"abc=null"},
			expected: `#
foo: bar
`,
		},
		{
			desc: "unset nested",
			inputs: []*yaml.RNode{yaml.MustParse(`#
foo: bar
abc:
  xyz: zyx
`)},
			specs: []string{"abc.xyz=null"},
			expected: `#
foo: bar
abc: {}
`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var buf strings.Builder
			err := kio.Pipeline{
				Inputs:  []kio.Reader{kio.ResourceNodeSlice(tc.inputs)},
				Filters: []kio.Filter{kio.FilterAll(SetValues(tc.specs, tc.forceString))},
				Outputs: []kio.Writer{kio.ByteWriter{Writer: &buf}},
			}.Execute()
			if assert.NoError(t, err, "Pipeline failed") {
				assert.YAMLEq(t, tc.expected, buf.String())
			}
		})
	}
}

func TestSplitPathValue(t *testing.T) {
	cases := []struct {
		desc          string
		spec          string
		expectedPath  string
		expectedValue *yaml.RNode
	}{
		{
			desc: "empty",
		},
		{
			desc:          "simple name string value",
			spec:          "name=value",
			expectedPath:  "name",
			expectedValue: yaml.NewStringRNode("value"),
		},
		{
			desc:          "nested name string value",
			spec:          "nom.name=value",
			expectedPath:  "nom.name",
			expectedValue: yaml.NewStringRNode("value"),
		},
		{
			desc:          "indexed name string value",
			spec:          "nom.name.[foo.bar=xyz].foobar=value",
			expectedPath:  "nom.name.[foo.bar=xyz].foobar",
			expectedValue: yaml.NewStringRNode("value"),
		},
		{
			desc:          "string value with delimiter",
			spec:          "name=Value. Such value.",
			expectedPath:  "name",
			expectedValue: yaml.NewStringRNode("Value. Such value."),
		},
		{
			desc:          "empty string value",
			spec:          "name=",
			expectedPath:  "name",
			expectedValue: yaml.NewStringRNode(""),
		},
		{
			desc:          "null value",
			spec:          "name=null",
			expectedPath:  "name",
			expectedValue: yaml.NewRNode(&yaml.Node{Kind: yaml.ScalarNode, Tag: yaml.NodeTagNull}),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			actualPath, actualValue := splitPathValue(tc.spec, false)
			assert.Equal(t, tc.expectedPath, actualPath)
			assert.Equal(t, tc.expectedValue, actualValue)
		})
	}
}
