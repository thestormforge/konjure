/*
Copyright 2021 GramLabs, Inc.

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

package konjure

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"text/template"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Writer is a multi-format writer for emitting resource nodes.
type Writer struct {
	// The desired format.
	Format string
	// The output stream to write to.
	Writer io.Writer
	// Flag to keep the intermediate annotations introduced during reading.
	KeepReaderAnnotations bool
	// List of additional annotations to clear.
	ClearAnnotations []string
	// Flag indicating nodes should be sorted before writing.
	Sort bool
}

// Write delegates to the format specific writer.
func (w *Writer) Write(nodes []*yaml.RNode) error {
	var ww kio.Writer
	switch strings.ToLower(w.Format) {

	case "yaml", "":
		ww = &kio.ByteWriter{
			Writer:                w.Writer,
			KeepReaderAnnotations: w.KeepReaderAnnotations,
			ClearAnnotations:      w.ClearAnnotations,
			Sort:                  w.Sort,
		}

	case "json":
		ww = &JSONWriter{
			Writer:                w.Writer,
			KeepReaderAnnotations: w.KeepReaderAnnotations,
			ClearAnnotations:      w.ClearAnnotations,
			WrappingAPIVersion:    "v1",
			WrappingKind:          "List",
			Sort:                  w.Sort,
		}

	case "ndjson":
		ww = &JSONWriter{
			Writer:                w.Writer,
			KeepReaderAnnotations: w.KeepReaderAnnotations,
			ClearAnnotations:      w.ClearAnnotations,
			Sort:                  w.Sort,
		}

	case "name":
		ww = &TemplateWriter{
			Writer:   w.Writer,
			Template: "{{ lower .kind }}/{{ .metadata.name }}\n",
		}

	case "env":
		ww = &EnvWriter{
			Writer: w.Writer,
		}

	default:
		if tmpl := columnsTemplate(w.Format); tmpl != "" {
			ww = &TemplateWriter{
				Writer:             tabwriter.NewWriter(w.Writer, 3, 0, 3, ' ', 0),
				WrappingAPIVersion: "v1",
				WrappingKind:       "List",
				Template:           tmpl,
			}
		}
	}

	if ww == nil {
		return fmt.Errorf("unknown format: %s", w.Format)
	}
	return ww.Write(nodes)
}

// JSONWriter is a writer which emits JSON instead of YAML. This is useful if you like `jq`.
type JSONWriter struct {
	Writer                io.Writer
	KeepReaderAnnotations bool
	ClearAnnotations      []string
	WrappingKind          string
	WrappingAPIVersion    string
	Sort                  bool
}

// Write encodes each node as a single line of JSON.
func (w *JSONWriter) Write(nodes []*yaml.RNode) error {
	if w.Sort {
		if err := kioutil.SortNodes(nodes); err != nil {
			return err
		}
	}

	enc := json.NewEncoder(w.Writer)
	for _, n := range nodes {
		// This is to be consistent with ByteWriter
		if !w.KeepReaderAnnotations {
			_, err := n.Pipe(yaml.ClearAnnotation(kioutil.IndexAnnotation))
			if err != nil {
				return err
			}
		}
		for _, a := range w.ClearAnnotations {
			_, err := n.Pipe(yaml.ClearAnnotation(a))
			if err != nil {
				return err
			}
		}
	}

	if w.WrappingKind == "" {
		for i := range nodes {
			if err := enc.Encode(nodes[i]); err != nil {
				return err
			}
		}
		return nil
	}

	return enc.Encode(wrap(w.WrappingAPIVersion, w.WrappingKind, nodes))
}

// TemplateWriter is a writer which emits each resource evaluated using a configured Go template.
type TemplateWriter struct {
	Writer             io.Writer
	Template           string
	Functions          template.FuncMap
	WrappingKind       string
	WrappingAPIVersion string
}

// Write evaluates the template using each resource.
func (w *TemplateWriter) Write(nodes []*yaml.RNode) error {
	fns := map[string]interface{}{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
	}
	for k, v := range w.Functions {
		fns[k] = v
	}

	tmpl, err := template.New("resource").Funcs(fns).Parse(w.Template)
	if err != nil {
		return err
	}

	if w.WrappingKind != "" {
		nodes = []*yaml.RNode{wrap(w.WrappingAPIVersion, w.WrappingKind, nodes)}
	}

	for _, n := range nodes {
		var data interface{}
		if err := n.YNode().Decode(&data); err != nil {
			return err
		}

		if err := tmpl.Execute(w.Writer, data); err != nil {
			return err
		}
	}

	if f, ok := w.Writer.(interface{ Flush() error }); ok {
		if err := f.Flush(); err != nil {
			return err
		}
	}

	return nil
}

// columnsTemplate transforms a format string into a Go template for generating a table,
// returns an empty string if conversion is not possible.
// NOTE: the resulting template is written against a "v1.List" wrapper.
func columnsTemplate(f string) string {
	cols := f
	cols = strings.TrimPrefix(cols, "columns=")
	cols = strings.TrimPrefix(cols, "cols=")
	if cols == f {
		return ""
	}

	var headers, columns []string
	for _, c := range strings.Split(cols, ",") {
		c = strings.TrimSpace(c)
		headers = append(headers, strings.ToUpper(c[strings.LastIndex(c, ".")+1:]))
		columns = append(columns, fmt.Sprintf("{{ .%s }}", strings.TrimPrefix(c, ".")))
	}

	return "{{ if .items }}" + strings.Join(headers, "\t") +
		"\n{{ range .items }}" + strings.Join(columns, "\t") +
		"\n{{ end }}{{ else }}No results.\n{{ end }}"
}

// EnvWriter is a writer which only emits name/value pairs found in the data of config maps and secrets.
type EnvWriter struct {
	Writer   io.Writer
	Unset    bool
	Shell    string
	Selector string
}

// Write outputs the data pairings from the supplied list of resource nodes.
func (w *EnvWriter) Write(nodes []*yaml.RNode) error {
	// Detect the shell from the environment
	sh := strings.ToLower(w.Shell)
	if sh == "" {
		if shell := os.Getenv("SHELL"); shell != "" {
			sh = strings.ToLower(filepath.Base(shell))
		}
	}

	for _, n := range nodes {
		if ok, err := n.MatchesLabelSelector(w.Selector); err == nil && !ok {
			continue
		}

		decode := func(s string) ([]byte, error) { return []byte(s), nil }
		if m, err := n.GetMeta(); err == nil && m.Kind == "Secret" {
			decode = base64.StdEncoding.DecodeString
		}

		for k, v := range n.GetDataMap() {
			b, err := decode(v)
			if err != nil {
				return err
			}
			v = string(b)

			// Assume this is file data and not simple name/value pairs
			if strings.Contains(k, ".") || strings.ContainsAny(v, "\n\r") {
				continue
			}

			// TODO Should we print a comment with the ID of the node the first time this hits?
			w.printEnvVar(sh, k, v)
		}
	}

	return nil
}

// printEnvVar emits a single pair.
func (w *EnvWriter) printEnvVar(sh, k, v string) {
	switch sh {
	case "none", "":
		if w.Unset {
			_, _ = fmt.Fprintf(w.Writer, "%s=\n", k)
		} else {
			_, _ = fmt.Fprintf(w.Writer, "%s=%s\n", k, v)
		}

	case "fish":
		// e.g.: SHELL=fish konjure --output env ... | source
		if w.Unset {
			_, _ = fmt.Fprintf(w.Writer, "set -e %s;\n", k)
		} else {
			_, _ = fmt.Fprintf(w.Writer, "set -gx %s %q;\n", k, v)
		}

	default: // sh, bash, zsh, etc.
		// e.g.: eval $(SHELL=zsh konjure --output env ...)
		if w.Unset {
			_, _ = fmt.Fprintf(w.Writer, "unset %s\n", k)
		} else {
			_, _ = fmt.Fprintf(w.Writer, "export %s=%q\n", k, v)
		}
	}
}

// wrap is a helper that wraps a list of resource nodes into a single node.
func wrap(apiVersion, kind string, nodes []*yaml.RNode) *yaml.RNode {
	items := &yaml.Node{Kind: yaml.SequenceNode}
	for i := range nodes {
		items.Content = append(items.Content, nodes[i].YNode())
	}

	return yaml.NewRNode(&yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind: yaml.MappingNode,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "apiVersion"},
					{Kind: yaml.ScalarNode, Value: apiVersion},
					{Kind: yaml.ScalarNode, Value: "kind"},
					{Kind: yaml.ScalarNode, Value: kind},
					{Kind: yaml.ScalarNode, Value: "items"},
					items,
				},
			},
		},
	})
}

// GroupWriter writes nodes based on a functional grouping definition.
type GroupWriter struct {
	KeepReaderAnnotations bool
	ClearAnnotations      []string
	GroupNode             func(node *yaml.RNode) (group string, ordinal string, err error)
	GroupWriter           func(name string) (io.Writer, error)
}

// Write sends all the output on the files back to where it came from.
func (w *GroupWriter) Write(nodes []*yaml.RNode) error {
	// Use the KYAML path/index annotations as the default grouping
	clearAnnotations := w.ClearAnnotations
	if w.GroupNode == nil {
		w.GroupNode = kioutil.GetFileAnnotations
		if !w.KeepReaderAnnotations {
			clearAnnotations = append(
				clearAnnotations,
				kioutil.PathAnnotation,
				kioutil.IndexAnnotation,
			)
		}
	}

	// Use os.Create for the default writer factory
	if w.GroupWriter == nil {
		w.GroupWriter = func(name string) (io.Writer, error) {
			if name == "" {
				return nil, nil
			}

			// This isn't very safe, but that's what file system permissions are for
			return os.Create(name)
		}
	}

	// Index the nodes
	indexed, err := w.indexNodes(nodes)
	if err != nil {
		return err
	}

	// Write each group
	for path, nodes := range indexed {
		// Get an io.Writer for the group
		out, err := w.GroupWriter(path)
		if err != nil {
			return err
		}
		if out == nil {
			continue
		}

		ww := &kio.ByteWriter{
			Writer:                out,
			KeepReaderAnnotations: w.KeepReaderAnnotations,
			ClearAnnotations:      clearAnnotations,
		}

		// Write the content out
		err = ww.Write(nodes)
		if c, ok := out.(io.Closer); ok {
			_ = c.Close()
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// indexNodes returns a sorted list of nodes indexed by group.
func (w *GroupWriter) indexNodes(nodes []*yaml.RNode) (map[string][]*yaml.RNode, error) {
	result := make(map[string][]*yaml.RNode)
	ordinal := make(map[string][]string)
	for i := range nodes {
		g, o, err := w.GroupNode(nodes[i])
		if err != nil {
			return nil, err
		}

		result[g] = append(result[g], nodes[i])
		ordinal[g] = append(ordinal[g], o)
	}

	// Sort the nodes using the ordinals we extracted (trying to preserve order)
	for group, nodes := range result {
		sort.SliceStable(nodes, func(i, j int) bool {
			// Try a pure numeric comparison first
			oi, erri := strconv.Atoi(ordinal[group][i])
			oj, errj := strconv.Atoi(ordinal[group][j])
			if erri == nil && errj == nil {
				return oi < oj
			}

			// Fall back to lexicographical ordering
			return ordinal[group][i] < ordinal[group][j]
		})
	}

	return result, nil
}
