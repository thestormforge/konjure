package pipes

import (
	"io/fs"
	"os"

	"github.com/thestormforge/konjure/pkg/pipes/internal/strvals"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// HelmValues is a reader that emits resource nodes representing Helm values.
type HelmValues struct {
	// User specified values files (via -f/--values).
	ValueFiles []string
	// User specified values (via --set).
	Values []string
	// User specified string values (via --set-string).
	StringValues []string
	// User specified file values (via --set-file).
	FileValues []string

	// The file system to use for resolving file contents (defaults to the OS reader).
	FS fs.FS
}

// Read converts the configured user specified values into resource nodes.
func (f *HelmValues) Read() ([]*yaml.RNode, error) {
	var result []*yaml.RNode

	for _, filePath := range f.ValueFiles {
		data, err := f.readFile(filePath)
		if err != nil {
			return nil, err
		}

		node, err := yaml.Parse(data)
		if err != nil {
			return nil, err
		}

		result = append(result, node)
	}

	for _, value := range f.Values {
		node, err := f.parse(value, strvals.Parse)
		if err != nil {
			return nil, err
		}
		result = append(result, node)
	}

	for _, value := range f.StringValues {
		node, err := f.parse(value, strvals.ParseString)
		if err != nil {
			return nil, err
		}
		result = append(result, node)
	}

	for _, value := range f.FileValues {
		node, err := f.parse(value, func(s string) (map[string]interface{}, error) {
			return strvals.ParseFile(s, func(rs []rune) (interface{}, error) { return f.readFile(string(rs)) })
		})
		if err != nil {
			return nil, err
		}
		result = append(result, node)
	}

	return result, nil
}

func (f *HelmValues) parse(v string, parseFunc func(string) (map[string]interface{}, error)) (*yaml.RNode, error) {
	data, err := parseFunc(v)
	if err != nil {
		return nil, err
	}

	node := yaml.NewRNode(&yaml.Node{})
	if err := node.YNode().Encode(data); err != nil {
		return nil, err
	}

	return node, nil
}

func (f *HelmValues) readFile(spec string) (string, error) {
	// TODO Should we be using something like spec.Parser to pull in data?

	if f.FS == nil {
		data, err := fs.ReadFile(f.FS, spec)
		return string(data), err
	}

	data, err := os.ReadFile(spec)
	return string(data), err
}
