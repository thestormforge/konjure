package filters

import (
	"io/fs"
	"os"

	"github.com/thestormforge/konjure/pkg/filters/internal/strvals"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"
)

// HelmValuesFilter is a filter that merges user supplied values into an incoming
// resource node.
type HelmValuesFilter struct {
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

// Filter merges the configured user specified values into the supplied node.
func (f *HelmValuesFilter) Filter(rn *yaml.RNode) (*yaml.RNode, error) {
	var filters []yaml.Filter

	for _, filePath := range f.ValueFiles {
		filters = append(filters, yaml.Tee(yaml.FilterFunc(func(dest *yaml.RNode) (*yaml.RNode, error) {
			data, err := f.readFile(filePath)
			if err != nil {
				return nil, err
			}

			if data == "" {
				return dest, nil
			}

			src, err := yaml.Parse(data)
			if err != nil {
				return nil, err
			}

			return merge2.Merge(src, dest, yaml.MergeOptions{})
		})))
	}

	for _, value := range f.Values {
		filters = append(filters, yaml.Tee(f.parseFilter(value, strvals.Parse)))
	}

	for _, value := range f.StringValues {
		filters = append(filters, yaml.Tee(f.parseFilter(value, strvals.ParseString)))
	}

	for _, value := range f.FileValues {
		filters = append(filters, yaml.Tee(f.parseFilter(value, func(s string) (map[string]interface{}, error) {
			return strvals.ParseFile(s, func(rs []rune) (interface{}, error) { return f.readFile(string(rs)) })
		})))
	}

	return rn.Pipe(filters...)
}

func (f *HelmValuesFilter) parseFilter(v string, parseFunc func(string) (map[string]interface{}, error)) yaml.Filter {
	return yaml.FilterFunc(func(dest *yaml.RNode) (*yaml.RNode, error) {
		m, err := parseFunc(v)
		if err != nil {
			return nil, err
		}

		src := yaml.NewRNode(&yaml.Node{})
		if err := src.YNode().Encode(m); err != nil {
			return nil, err
		}

		return merge2.Merge(src, dest, yaml.MergeOptions{})
	})
}

func (f *HelmValuesFilter) readFile(spec string) (string, error) {
	// TODO Should we be using something like spec.Parser to pull in data?

	if f.FS == nil {
		data, err := fs.ReadFile(f.FS, spec)
		return string(data), err
	}

	data, err := os.ReadFile(spec)
	return string(data), err
}
