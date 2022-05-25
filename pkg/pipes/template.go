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
