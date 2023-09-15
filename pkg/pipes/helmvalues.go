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

package pipes

import (
	"io/fs"
	"os"

	"github.com/thestormforge/konjure/pkg/pipes/internal/strvals"
	"sigs.k8s.io/kustomize/kyaml/kio"
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

// AsMap converts the configured user specified values into a map of values.
func (r *HelmValues) AsMap() (map[string]any, error) {
	base := map[string]any{}

	for _, filePath := range r.ValueFiles {
		currentMap := map[string]any{}

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
		if err := strvals.ParseIntoFile(value, base, func(rs []rune) (any, error) { return r.readFile(string(rs)) }); err != nil {
			return nil, err
		}
	}

	if len(base) == 0 {
		return nil, nil
	}
	return base, nil
}

// Read converts the configured user specified values into resource nodes.
func (r *HelmValues) Read() ([]*yaml.RNode, error) {
	base, err := r.AsMap()
	if err != nil {
		return nil, err
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
func (r *HelmValues) MergeMaps(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]any); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]any); ok {
					out[k] = r.MergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

// Mask returns a filter that either keeps or strips data impacted by these values.
func (r *HelmValues) Mask(keep bool) kio.Filter {
	return kio.FilterFunc(func(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
		m, err := r.AsMap()
		if err != nil {
			return nil, err
		}

		result := make([]*yaml.RNode, 0, len(nodes))
		for _, n := range nodes {
			if nn, err := mask(n, m, keep); err != nil {
				return nil, err
			} else if nn != nil {
				result = append(result, nn)
			}
		}
		return result, nil
	})
}

func mask(rn *yaml.RNode, m any, keep bool) (*yaml.RNode, error) {
	switch m := m.(type) {
	case map[string]any:
		if err := yaml.ErrorIfInvalid(rn, yaml.MappingNode); err != nil {
			return nil, err
		}

		original := rn.Content()
		masked := make([]*yaml.Node, 0, len(original))
		for i := 0; i < len(original); i += 2 {
			if v, ok := m[original[i].Value]; ok {
				// Recursively filter the value
				if value, err := mask(yaml.NewRNode(original[i+1]), v, keep); err != nil {
					return nil, err
				} else if value != nil {
					masked = append(masked, original[i], value.YNode())
				}
			} else if !keep {
				// Just keep it
				masked = append(masked, original[i], original[i+1])
			}
		}
		if len(masked) > 0 {
			rn = rn.Copy()
			rn.YNode().Content = masked
			return rn, nil
		}

	default:
		if keep && m != nil {
			return rn.Copy(), nil
		}
	}
	return nil, nil
}
