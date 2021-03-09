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
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
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
	var clearAnnotations []string
	clearAnnotations = append(clearAnnotations, w.ClearAnnotations...)
	if !w.KeepReaderAnnotations {
		clearAnnotations = append(clearAnnotations,
			kioutil.PathAnnotation,
			filters.FmtAnnotation,
		)
	}

	var ww kio.Writer
	switch strings.ToLower(w.Format) {

	case "yaml", "":
		ww = &kio.ByteWriter{
			Writer:                w.Writer,
			KeepReaderAnnotations: w.KeepReaderAnnotations,
			ClearAnnotations:      clearAnnotations,
			Sort:                  w.Sort,
		}

	case "ndjson", "json":
		ww = &NDJSONWriter{
			Writer:                w.Writer,
			KeepReaderAnnotations: w.KeepReaderAnnotations,
			ClearAnnotations:      clearAnnotations,
			Sort:                  w.Sort,
		}

	case "env":
		ww = &EnvWriter{
			Writer: w.Writer,
		}

	default:
		return fmt.Errorf("unknown format: %s", w.Format)
	}

	return ww.Write(nodes)
}

// NDJSONWriter is a writer which emits JSON instead of YAML. This is useful if you like jq.
type NDJSONWriter struct {
	Writer                io.Writer
	KeepReaderAnnotations bool
	ClearAnnotations      []string
	Sort                  bool
}

// Write encodes each node as a single line of JSON.
func (w *NDJSONWriter) Write(nodes []*yaml.RNode) error {
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

		if err := enc.Encode(n); err != nil {
			return err
		}
	}

	return nil
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
			w.printEnvVar(k, v)
		}
	}

	return nil
}

// printEnvVar emits a single pair.
func (w *EnvWriter) printEnvVar(k, v string) {
	switch w.Shell {
	case "":
		if w.Unset {
			_, _ = fmt.Fprintf(w.Writer, "%s=\n", k)
		} else {
			_, _ = fmt.Fprintf(w.Writer, "%s=%s\n", k, v)
		}

	default:
		if w.Unset {
			_, _ = fmt.Fprintf(w.Writer, "unset %s\n", k)
		} else {
			_, _ = fmt.Fprintf(w.Writer, "export %s=%q\n", k, v)
		}
	}
}
