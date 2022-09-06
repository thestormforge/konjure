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
	base := map[string]interface{}{}

	for _, filePath := range f.ValueFiles {
		currentMap := map[string]interface{}{}

		data, err := f.readFile(filePath)
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal([]byte(data), &currentMap); err != nil {
			return nil, err
		}

		base = mergeMaps(base, currentMap)
	}

	for _, value := range f.Values {
		if err := strvals.ParseInto(value, base); err != nil {
			return nil, err
		}
	}

	for _, value := range f.StringValues {
		if err := strvals.ParseIntoString(value, base); err != nil {
			return nil, err
		}
	}

	for _, value := range f.FileValues {
		if err := strvals.ParseIntoFile(value, base, func(rs []rune) (interface{}, error) { return f.readFile(string(rs)) }); err != nil {
			return nil, err
		}
	}

	if len(base) == 0 {
		return nil, nil
	}

	node := yaml.NewRNode(&yaml.Node{})
	if err := node.YNode().Encode(base); err != nil {
		return nil, err
	}
	return []*yaml.RNode{node}, nil
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

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}
