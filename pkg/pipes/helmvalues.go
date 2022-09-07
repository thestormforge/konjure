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
func (r *HelmValues) Read() ([]*yaml.RNode, error) {
	base := map[string]interface{}{}

	for _, filePath := range r.ValueFiles {
		currentMap := map[string]interface{}{}

		data, err := r.readFile(filePath)
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal([]byte(data), &currentMap); err != nil {
			return nil, err
		}

		base = r.MergeMaps(base, currentMap)
	}

	for _, value := range r.Values {
		if err := strvals.ParseInto(value, base); err != nil {
			return nil, err
		}
	}

	for _, value := range r.StringValues {
		if err := strvals.ParseIntoString(value, base); err != nil {
			return nil, err
		}
	}

	for _, value := range r.FileValues {
		if err := strvals.ParseIntoFile(value, base, func(rs []rune) (interface{}, error) { return r.readFile(string(rs)) }); err != nil {
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

func (r *HelmValues) readFile(spec string) (string, error) {
	// TODO Should we be using something like spec.Parser to pull in data?

	if r.FS != nil {
		data, err := fs.ReadFile(r.FS, spec)
		return string(data), err
	}

	data, err := os.ReadFile(spec)
	return string(data), err
}

// MergeMaps is used to combine results from multiple values.
func (r *HelmValues) MergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = r.MergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}
