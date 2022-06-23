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
