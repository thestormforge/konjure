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
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Writer is a multi-format writer for emitting resource nodes.
type Writer struct {
	// The output stream to write to.
	Writer io.Writer
	// The desired format.
	Format string
	// Flag indicating nodes should be sorted before writing.
	Sort bool
}

// Write delegates to the format specific writer.
func (w *Writer) Write(nodes []*yaml.RNode) error {
	var ww kio.Writer
	switch strings.ToLower(w.Format) {

	case "yaml", "":
		ww = &kio.ByteWriter{
			Writer: w.Writer,
			// Do not set 'Sort' here so we can handle the sort for all writers
		}

	case "json":
		ww = &NDJSONWriter{
			Writer: w.Writer,
		}

	case "env":
		ww = &EnvWriter{
			Writer: w.Writer,
		}

	default:
		return fmt.Errorf("unknown format: %s", w.Format)
	}

	// Sort the nodes prior to writing if necessary
	if w.Sort {
		if err := kioutil.SortNodes(nodes); err != nil {
			return err
		}
	}

	return ww.Write(nodes)
}

// NDJSONWriter is a writer which emits JSON instead of YAML. This is useful if you like jq.
type NDJSONWriter struct {
	Writer io.Writer
}

// Write encodes each node as a single line of JSON.
func (w *NDJSONWriter) Write(nodes []*yaml.RNode) error {
	enc := json.NewEncoder(w.Writer)
	for _, n := range nodes {
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
