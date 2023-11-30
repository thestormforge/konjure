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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/thestormforge/konjure/pkg/filters"
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
	// Flag indicating we should attempt to restore vertical white space using
	// line numbers prior to writing YAML output.
	RestoreVerticalWhiteSpace bool
	// Normally, document start indicators are only included between resources.
	InitialDocumentStart bool
	// The default Go template to evaluate. Alternately, when the format is "env", a glob matching the file name to emit.
	Template string
	// The root template used to parse user supplied templates. Can be used to
	// inject new functions or templates.
	RootTemplate *template.Template
	// Generic configuration options for specific writer implementations.
	Options []WriterOption
}

// WriterOption is an option for specific writer implementations.
type WriterOption func(kio.Writer)

// Write delegates to the format specific writer.
func (w *Writer) Write(nodes []*yaml.RNode) error {
	var ww kio.Writer

	// TODO This is a hack for detecting when the writer is being used for non-Kube resources
	var nonKube bool
	if len(nodes) > 0 {
		meta, _ := nodes[0].GetMeta()
		nonKube = meta.Kind == ""
	}

	// Determine the effective format and template
	f, t := strings.ToLower(w.Format), w.Template
	if pos := strings.IndexRune(f, '=') + 1; pos > 0 {
		f, t = f[0:pos-1], w.Format[pos:]
	} else if strings.Contains(f, "{{") {
		f, t = "template", w.Format
	}

	switch f {

	case "yaml", "":
		if w.RestoreVerticalWhiteSpace {
			restoreVerticalWhiteSpace(nodes)
		}

		ww = &kio.ByteWriter{
			Writer:                w.Writer,
			KeepReaderAnnotations: w.KeepReaderAnnotations,
			ClearAnnotations:      w.ClearAnnotations,
		}

		// The ByteWriter will not print the first document start indicator
		if len(nodes) > 0 && w.InitialDocumentStart && !nonKube {
			if _, err := w.Writer.Write([]byte("---\n")); err != nil {
				return err
			}
		}

	case "json":
		ww = &JSONWriter{
			Writer:                w.Writer,
			KeepReaderAnnotations: w.KeepReaderAnnotations,
			ClearAnnotations:      w.ClearAnnotations,
			WrappingAPIVersion:    "v1",
			WrappingKind:          "List",
		}

		// Allow some JSON to sneak through unwrapped
		if len(nodes) == 1 && nonKube {
			ww.(*JSONWriter).WrappingAPIVersion = ""
			ww.(*JSONWriter).WrappingKind = ""
		}

	case "ndjson":
		ww = &JSONWriter{
			Writer:                w.Writer,
			KeepReaderAnnotations: w.KeepReaderAnnotations,
			ClearAnnotations:      w.ClearAnnotations,
		}

	case "env":
		ww = &EnvWriter{
			Writer:      w.Writer,
			FilePattern: t,
		}

	case "name":
		ww = &TemplateWriter{
			Writer:   w.Writer,
			Template: "{{ lower .kind }}/{{ .metadata.name }}\n",
		}

	case "template", "go-template":
		ww = &TemplateWriter{
			Writer:       w.Writer,
			RootTemplate: w.RootTemplate,
			Template:     t,
		}

	case "columns", "custom-columns":
		headers, columns := splitColumns(t)
		for i := range columns {
			columns[i] = fmt.Sprintf("{{ index . %q }}", strings.TrimPrefix(columns[i], "."))
		}

		ww = &TemplateWriter{
			Writer:             tabwriter.NewWriter(w.Writer, 3, 0, 3, ' ', 0),
			RootTemplate:       w.RootTemplate,
			WrappingAPIVersion: "v1",
			WrappingKind:       "List",
			Template: "{{ if .items }}" + strings.Join(headers, "\t") +
				"\n{{ range .items }}" + strings.Join(columns, "\t") +
				"\n{{ end }}{{ else }}No results.\n{{ end }}",
		}

	case "csv":
		headers, paths := splitColumns(t)
		columns := make([][]string, 0, len(paths))
		for _, p := range paths {
			column, err := filters.FieldPath(p, nil)
			if err != nil {
				return err
			}
			columns = append(columns, column)
		}

		ww = &CSVWriter{
			Writer:  w.Writer,
			Headers: headers,
			Columns: columns,
		}

	}

	if ww == nil {
		return fmt.Errorf("unknown format: %s", w.Format)
	}
	for _, opt := range w.Options {
		opt(ww)
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
			if err := n.PipeE(
				yaml.ClearAnnotation(kioutil.IndexAnnotation),
				yaml.ClearAnnotation(kioutil.LegacyIndexAnnotation),
				yaml.ClearAnnotation(kioutil.SeqIndentAnnotation),
			); err != nil {
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
	RootTemplate       *template.Template
	WrappingKind       string
	WrappingAPIVersion string
}

// Write evaluates the template using each resource.
func (w *TemplateWriter) Write(nodes []*yaml.RNode) error {
	root := w.RootTemplate
	if root == nil {
		root = template.New("root").Funcs(template.FuncMap{
			"upper": strings.ToUpper,
			"lower": strings.ToLower,
		})
	}

	tmpl, err := root.New("resource").Parse(w.Template)
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

// CSVWriter is a writer which emits comma-separated values based on the supplied column paths.
type CSVWriter struct {
	Writer  io.Writer
	Headers []string
	Columns [][]string
}

// Write outputs the data as CSV.
func (w *CSVWriter) Write(nodes []*yaml.RNode) error {
	cw := csv.NewWriter(w.Writer)
	if len(w.Headers) > 0 {
		if err := cw.Write(w.Headers); err != nil {
			return err
		}
	}

	record := make([]string, len(w.Columns))
	for _, node := range nodes {
		for i, col := range w.Columns {
			c, err := node.Pipe(yaml.Lookup(col...))
			if err != nil {
				return err
			}

			// TODO How should we convert this to string?
			record[i] = c.YNode().Value
		}

		if err := cw.Write(record); err != nil {
			return err
		}
	}

	cw.Flush()
	return cw.Error()
}

// EnvWriter is a writer which only emits name/value pairs found in the data of config maps and secrets.
type EnvWriter struct {
	Writer      io.Writer
	Unset       bool
	Shell       string
	Selector    string
	FilePattern string
	Comments    bool
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
		// Only consider matching nodes
		if ok, err := n.MatchesLabelSelector(w.Selector); err == nil && !ok {
			continue
		}

		md, err := n.GetMeta()
		if err != nil {
			continue
		}

		var dataMap map[string]string
		switch {
		case md.Kind == "ConfigMap":
			dataMap = n.GetDataMap()

			// Ignore the binaryData field unless we are looking for files
			if w.FilePattern != "" {
				for k, v := range n.GetBinaryDataMap() {
					dataMap[k] = v
				}
			}

		case md.Kind == "Secret":
			dataMap = n.GetDataMap()

			// Decode the secret data
			for k, v := range dataMap {
				if vv, err := base64.StdEncoding.DecodeString(v); err == nil {
					dataMap[k] = string(vv)
				}
			}

			// Since we might be looking at raw YAML, also consider the stringData field
			_ = n.PipeE(yaml.Lookup("stringData"), yaml.FilterFunc(func(object *yaml.RNode) (*yaml.RNode, error) {
				return nil, object.VisitFields(func(node *yaml.MapNode) error {
					dataMap[yaml.GetValue(node.Key)] = yaml.GetValue(node.Value)
					return nil
				})
			}))

		default:
			dataMap = map[string]string{}

			// Look for container environment variables that use `value` (i.e. not `valueFrom`)
			_ = n.PipeE(
				yaml.LookupFirstMatch(yaml.ConventionalContainerPaths),
				&yaml.PathMatcher{Path: []string{"[name=]", "env", "[value=.+]"}},
				yaml.FilterFunc(func(object *yaml.RNode) (*yaml.RNode, error) {
					return nil, object.VisitElements(func(node *yaml.RNode) error {
						name, err := node.GetString("name")
						if err != nil {
							return err
						}
						value, err := node.GetString("value")
						if err != nil {
							return err
						}
						dataMap[name] = value
						return nil
					})
				}))
		}

		// Decode and print each value from the map
		if len(dataMap) > 0 {
			sortedKeys := make([]string, 0, len(dataMap))
			for k := range dataMap {
				sortedKeys = append(sortedKeys, k)
			}
			sort.Strings(sortedKeys)

			_, _ = w.printComment(sh, fmt.Sprintf("%s %s", md.Kind, md.Name))
			for _, k := range sortedKeys {
				_, _ = w.printEnvVar(sh, k, dataMap[k])
			}
		}
	}

	return nil
}

// printComment emits a comment.
func (w *EnvWriter) printComment(sh, c string) (int, error) {
	switch {
	case !w.Comments:
		// Comments are disabled
		return 0, nil
	}

	switch sh {
	case "none", "":
		return 0, nil
	default: // sh, bash, zsh, fish, etc.
		return fmt.Fprintf(w.Writer, "# %s\n", c)
	}
}

// printEnvVar emits a single pair.
func (w *EnvWriter) printEnvVar(sh, k, v string) (int, error) {
	switch {
	case w.FilePattern != "":
		// If we have a file pattern specified, we must match it
		if ok, err := path.Match(w.FilePattern, k); err == nil && ok {
			return fmt.Fprint(w.Writer, v)
		}
		return 0, nil

	case strings.ContainsAny(v, "\n\r"):
		// If the value contains newlines, it is most likely "file content", do not emit anything
		return 0, nil

	case strings.Contains(k, "."):
		// If the key contains a dot, it is most likely a file name, not an environment variable
		return 0, nil
	}

	switch sh {
	case "none", "":
		if w.Unset {
			return fmt.Fprintf(w.Writer, "%s=\n", k)
		} else {
			return fmt.Fprintf(w.Writer, "%s=%s\n", k, v)
		}

	case "fish":
		// e.g.: SHELL=fish konjure --output env ... | source
		if w.Unset {
			return fmt.Fprintf(w.Writer, "set -e %s;\n", k)
		} else {
			return fmt.Fprintf(w.Writer, "set -gx %s %q;\n", k, v)
		}

	default: // sh, bash, zsh, etc.
		// e.g.: eval $(SHELL=zsh konjure --output env ...)
		if w.Unset {
			return fmt.Fprintf(w.Writer, "unset %s\n", k)
		} else {
			return fmt.Fprintf(w.Writer, "export %s=%q\n", k, v)
		}
	}
}

// extractDockerConfig makes it easier to read Docker configurations. You can bypass this
// expansion by setting the FilePattern to `.dockerconfigjson` (i.e. emit the raw file).
func (w *EnvWriter) extractDockerConfig(n *yaml.RNode) map[string]string {
	if w.FilePattern != "" {
		return nil
	}
	if t, err := n.GetString("type"); err != nil || t != "kubernetes.io/dockerconfigjson" {
		return nil
	}
	configJSON, err := base64.StdEncoding.DecodeString(n.GetDataMap()[".dockerconfigjson"])
	if err != nil {
		return nil
	}
	config := struct {
		Auths map[string]struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Auth     string `json:"auth"`
		} `json:"auths"`
	}{}
	if err := json.Unmarshal(configJSON, &config); err != nil || len(config.Auths) != 1 {
		return nil
	}
	for k, v := range config.Auths {
		return map[string]string{
			"DOCKER_REGISTRY": base64.StdEncoding.EncodeToString([]byte(k)),
			"DOCKER_USERNAME": base64.StdEncoding.EncodeToString([]byte(v.Username)),
			"DOCKER_PASSWORD": base64.StdEncoding.EncodeToString([]byte(v.Password)),
			"DOCKER_AUTH":     base64.StdEncoding.EncodeToString([]byte(v.Auth)),
		}
	}
	return nil
}

// GroupWriter writes nodes based on a functional grouping definition.
type GroupWriter struct {
	GroupNode   func(node *yaml.RNode) (group string, ordinal string, err error)
	GroupWriter func(name string) (io.Writer, error)

	KeepReaderAnnotations     bool
	ClearAnnotations          []string
	Sort                      bool
	RestoreVerticalWhiteSpace bool
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

				// Also remove the "legacy" variants
				"config.kubernetes.io/path",
				"config.kubernetes.io/index",
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

	// Attempt to restore vertical white space
	if w.RestoreVerticalWhiteSpace {
		restoreVerticalWhiteSpace(nodes)
	}

	// Index the nodes
	indexed, err := w.indexNodes(nodes)
	if err != nil {
		return err
	}

	// Write each group
	for name, nodes := range indexed {
		// Get an io.Writer for the group
		out, err := w.GroupWriter(name)
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
			Sort:                  w.Sort,
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

// restoreVerticalWhiteSpace tries to put back blank lines eaten by the parser.
// It's not perfect (it only restores blank lines on the top level), but it helps
// prevent some changes to YAML sources that contain extra blank lines.
func restoreVerticalWhiteSpace(nodes []*yaml.RNode) {
	for _, node := range nodes {
		n := node.YNode()
		minLL := n.Line
		for i := range n.Content {
			// No need to insert VWS if we are still on the same line
			if i == 0 || n.Content[i].Line == n.Content[i-1].Line {
				continue
			}

			// Assume all lines before this node's head comment are blank and work back from there
			ll := n.Content[i].Line - 1
			if len(n.Content[i].HeadComment) > 0 {
				ll -= strings.Count(n.Content[i].HeadComment, "\n") + 1
			}

			// The previous node will have accounted for all the blanks above it
			if cll := lastLine(n.Content[i-1]); cll > minLL {
				minLL = cll
			}
			ll -= minLL

			// The foot comment will be stored two nodes back if this is a mapping node
			footComment := n.Content[i-1].FootComment
			if footComment == "" && n.Kind == yaml.MappingNode && i-2 >= 0 {
				footComment = n.Content[i-2].FootComment
			}
			if len(footComment) > 0 {
				ll -= strings.Count(footComment, "\n") + 2
			}

			// Check if all the lines are accounted for
			if ll <= 0 {
				continue
			}

			// Prefix the head comment with blank lines
			n.Content[i].HeadComment = strings.Repeat("\n", ll) + n.Content[i].HeadComment
		}
	}
}

// lastLine returns the largest line number from the supplied node.
func lastLine(n *yaml.Node) int {
	line := n.Line
	for i := range n.Content {
		if ll := lastLine(n.Content[i]); ll > line {
			line = ll
		}
	}
	return line
}

// splitColumns splits a column specification into fields, also returning the header names.
func splitColumns(spec string) (headers []string, columns []string) {
	for _, c := range strings.Split(spec, ",") {
		c = strings.TrimSpace(c)
		if pos := strings.IndexRune(c, ':'); pos > 0 {
			headers = append(headers, c[0:pos])
			columns = append(columns, c[pos+1:])
		} else {
			if pos == 0 {
				c = c[1:]
			}
			headers = append(headers, strings.ToUpper(c[strings.LastIndex(c, ".")+1:]))
			columns = append(columns, c)
		}
	}
	return
}
