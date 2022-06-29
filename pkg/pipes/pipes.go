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
	"bytes"
	"text/template"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ReaderFunc is an adapter to allow the use of ordinary functions as a kio.Reader.
type ReaderFunc func() ([]*yaml.RNode, error)

// Read evaluates the typed function.
func (r ReaderFunc) Read() ([]*yaml.RNode, error) { return r() }

// ReadOneFunc is an adapter to allow the use of single node returning functions as a kio.Reader.
type ReadOneFunc func() (*yaml.RNode, error)

// Read evaluates the typed function and wraps the resulting non-nil node.
func (r ReadOneFunc) Read() ([]*yaml.RNode, error) {
	node, err := r()
	if node != nil {
		return []*yaml.RNode{node}, err
	}
	return nil, err
}

// ErrorReader is an adapter to allow the use of an error as a kio.Reader.
type ErrorReader struct{ Err error }

// Reader returns the wrapped failure.
func (r ErrorReader) Read() ([]*yaml.RNode, error) { return nil, r.Err }

// EncodingReader is an adapter to allow arbitrary values to be used as a kio.Reader.
type EncodingReader struct{ Values []interface{} }

// Read encodes the configured values.
func (r *EncodingReader) Read() ([]*yaml.RNode, error) {
	result := make([]*yaml.RNode, len(r.Values))
	for i := range r.Values {
		result[i] = yaml.NewRNode(&yaml.Node{})
		if err := result[i].YNode().Encode(r.Values[i]); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// TemplateReader is a KYAML reader that consumes YAML from a Go template.
type TemplateReader struct {
	// The template to execute.
	Template *template.Template
	// The data for the template.
	Data interface{}
}

// Read executes the supplied template and parses the output as a YAML document stream.
func (c *TemplateReader) Read() ([]*yaml.RNode, error) {
	var buf bytes.Buffer
	if err := c.Template.Execute(&buf, c.Data); err != nil {
		return nil, err
	}

	return (&kio.ByteReader{
		Reader: &buf,
	}).Read()
}
