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
	"os/exec"
	"time"

	"github.com/thestormforge/konjure/pkg/tracing"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ExecReader is a KYAML reader that consumes YAML from another process via stdout.
type ExecReader struct {
	// The YAML producing command.
	*exec.Cmd
}

// Read executes the supplied command and parses the output as a YAML document stream.
func (c *ExecReader) Read() ([]*yaml.RNode, error) {
	start := time.Now()
	defer tracing.Exec(c.Cmd, start)
	data, err := c.Cmd.Output()
	if err != nil {
		return nil, err
	}

	return (&kio.ByteReader{
		Reader: bytes.NewReader(data),
	}).Read()
}

// ExecWriter is a KYAML writer that sends YAML to another process via stdin.
type ExecWriter struct {
	// The YAML consuming command.
	*exec.Cmd
}

// Write executes the supplied command, piping the generated YAML to stdin.
func (c *ExecWriter) Write(nodes []*yaml.RNode) error {
	// Open stdin for writing
	p, err := c.Cmd.StdinPipe()
	if err != nil {
		return err
	}

	// Start the command
	start := time.Now()
	if err := c.Cmd.Start(); err != nil {
		return err
	}

	// Sync on the command finishing and/or the byte writer writing
	g := errgroup.Group{}
	g.Go(func() error {
		defer tracing.Exec(c.Cmd, start)
		return c.Cmd.Wait()
	})
	g.Go(func() error {
		defer p.Close()
		return kio.ByteWriter{
			Writer: p, // TODO Use `bufio.NewWriter(p)`?
		}.Write(nodes)
	})
	return g.Wait()
}
