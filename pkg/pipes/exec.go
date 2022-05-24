package pipes

import (
	"bytes"
	"os/exec"

	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// CommandReader is a KYAML reader that consumes YAML from another process via stdout.
type CommandReader struct {
	// The YAML producing command.
	*exec.Cmd
}

// Read executes the supplied command and parses the output as a YAML document stream.
func (c *CommandReader) Read() ([]*yaml.RNode, error) {
	data, err := c.Cmd.Output()
	if err != nil {
		return nil, err
	}

	return (&kio.ByteReader{
		Reader: bytes.NewReader(data),
	}).Read()
}

// CommandWriter is a KYAML writer that sends YAML to another process via stdin.
type CommandWriter struct {
	// The YAML consuming command.
	*exec.Cmd
}

// Write executes the supplied command, piping the generated YAML to stdin.
func (c *CommandWriter) Write(nodes []*yaml.RNode) error {
	// Open stdin for writing
	p, err := c.Cmd.StdinPipe()
	if err != nil {
		return err
	}

	// Start the command
	if err := c.Cmd.Start(); err != nil {
		return err
	}

	// Sync on the command finishing and/or the byte writer writing
	g := errgroup.Group{}
	g.Go(c.Cmd.Wait)
	g.Go(func() error {
		defer p.Close()
		return kio.ByteWriter{
			Writer: p, // TODO Use `bufio.NewWriter(p)`?
		}.Write(nodes)
	})
	return g.Wait()
}
