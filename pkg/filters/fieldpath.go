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

package filters

import (
	"strconv"
	"strings"
	"text/template"

	"sigs.k8s.io/kustomize/kyaml/utils"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// FieldPath evaluates a path template using the supplied context and then
// splits it into individual path segments (honoring escaped delimiters).
func FieldPath(p string, data map[string]string) ([]string, error) {
	// Evaluate the path as a Go Template
	t, err := template.New("path").
		Delims("{", "}").
		Option("missingkey=zero").
		Parse(p)
	if err != nil {
		return nil, err
	}

	var pathBuf strings.Builder
	if err := t.Execute(&pathBuf, data); err != nil {
		return nil, err
	}

	return cleanPath(utils.SmarterPathSplitter(pathBuf.String(), "/")), nil
}

// SetPath returns a filter that sets the node value at the specified path.
// Note that this path uses "." separators rather then "/".
func SetPath(p string, v *yaml.RNode) yaml.Filter {
	path := cleanPath(utils.SmarterPathSplitter(p, "."))

	var fns []yaml.Filter
	if yaml.IsMissingOrNull(v) {
		if l := len(path) - 1; l == 0 {
			fns = append(fns,
				&yaml.FieldClearer{Name: path[l]},
			)
		} else {
			fns = append(fns,
				&yaml.PathGetter{Path: path[0:l]},
				&yaml.FieldClearer{Name: path[l], IfEmpty: true},
			)
		}
	} else {
		fns = append(fns,
			&yaml.PathGetter{Path: path, Create: v.YNode().Kind},
			&yaml.FieldSetter{Value: v},
		)
	}

	return yaml.FilterFunc(func(object *yaml.RNode) (*yaml.RNode, error) { return object.Pipe(fns...) })
}

// SetValues returns a filter that sets the supplied "path=value" specifications
// onto incoming nodes. Use forceString to bypass tagging the node values.
func SetValues(nameValue []string, forceString bool) yaml.Filter {
	var fns []yaml.Filter
	for _, spec := range nameValue {
		fns = append(fns, SetPath(splitPathValue(spec, forceString)))
	}

	return yaml.FilterFunc(func(object *yaml.RNode) (*yaml.RNode, error) { return object.Pipe(fns...) })
}

// cleanPath removes all empty and white space path elements.
func cleanPath(path []string) []string {
	result := make([]string, 0, len(path))
	for _, p := range path {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	if len(result) > 0 {
		return result
	}
	return nil
}

// splitPathValue splits a "path=value" into a path and an RNode.
func splitPathValue(spec string, st bool) (string, *yaml.RNode) {
	p := utils.SmarterPathSplitter(spec, ".")
	for i := len(p) - 1; i >= 0; i-- {
		path, value, ok := strings.Cut(p[i], "=")
		if !ok {
			continue
		}

		p[i] = path
		path = strings.Join(p[0:i+1], ".")

		p[i] = value
		value = strings.Join(p[i:], ".")

		node := yaml.NewStringRNode(value)
		if st {
			return path, node
		}

		switch strings.ToLower(value) {
		case "true":
			node.YNode().Tag = yaml.NodeTagBool
		case "false":
			node.YNode().Tag = yaml.NodeTagBool
		case "null":
			node.YNode().Tag = yaml.NodeTagNull
			node.YNode().Value = ""
		case "0":
			node.YNode().Tag = yaml.NodeTagInt
		default:
			if _, err := strconv.ParseInt(value, 10, 64); err == nil && value[0] != '0' {
				node.YNode().Tag = yaml.NodeTagInt
			}
		}
		return path, node
	}
	return spec, nil
}
