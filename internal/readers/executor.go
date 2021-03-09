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

package readers

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Executor is function that returns the output of a command.
type Executor func(cmd *exec.Cmd) ([]byte, error)

// WithCommandExecutor allows execution of an external process to be controlled
// by the specified alternate executor.
func WithCommandExecutor(cmd string, executor Executor) Option {
	return func(r kio.Reader) kio.Reader {
		return useExecutor(r, cmd, executor)
	}
}

// useExecutor wraps executable readers matching the supplied base command name
// with an explicit executor.
func useExecutor(r kio.Reader, cmd string, executor Executor) kio.Reader {
	switch er := r.(type) {

	case *Pipeline:
		for i := range er.Inputs {
			er.Inputs[i] = useExecutor(er.Inputs[i], cmd, executor)
		}

	case *ExecReader:
		if filepath.Base(er.Path) == cmd {
			return &executableReader{
				Command:  (*exec.Cmd)(er),
				Executor: executor,
			}
		}
	}

	return r
}

// FromCommand returns the resource nodes parsed from the output of an executable
// command. If the supplied executor is nil, cmd.Output will be used.
func FromCommand(cmd *exec.Cmd, executor Executor) ([]*yaml.RNode, error) {
	var out []byte
	var err error
	if executor != nil {
		out, err = executor(cmd)
	} else {
		out, err = cmd.Output()
	}

	// Try to clean up exit errors with a little bit of context
	if eerr, ok := err.(*exec.ExitError); ok {
		msg := strings.TrimSpace(string(eerr.Stderr))
		msg = strings.TrimPrefix(msg, "Error: ")
		return nil, fmt.Errorf("%s %w: %s", filepath.Base(cmd.Path), err, msg)
	}

	if err != nil {
		return nil, err
	}

	return kio.FromBytes(out)
}

// ExecReader allows an executable command to be used as a kio.Reader
type ExecReader exec.Cmd

// Read will buffer the output of the supplied command and parse it as resource nodes.
func (cmd *ExecReader) Read() ([]*yaml.RNode, error) {
	return FromCommand((*exec.Cmd)(cmd), nil)
}

// executableReader is like ExecReader but with an explicit executor.
type executableReader struct {
	Command  *exec.Cmd
	Executor Executor
}

// Read will buffer the output of the supplied command and parse it as resource nodes.
func (e *executableReader) Read() ([]*yaml.RNode, error) {
	return FromCommand(e.Command, e.Executor)
}

// Pipeline wraps a KYAML pipeline but doesn't allow writers: instead the
// resulting resource nodes are returned directly. This is useful for applying
// filters to readers in memory. A pipeline can also be used as a reader in
// larger pipelines.
type Pipeline struct {
	Inputs                []kio.Reader
	Filters               []kio.Filter
	ContinueOnEmptyResult bool
}

// Execute this pipeline, returning the resulting resource nodes directly.
func (p *Pipeline) Execute() ([]*yaml.RNode, error) {
	var result []*yaml.RNode

	pp := kio.Pipeline{
		Inputs:                p.Inputs,
		Filters:               p.Filters,
		ContinueOnEmptyResult: p.ContinueOnEmptyResult,
		Outputs: []kio.Writer{kio.WriterFunc(func(nodes []*yaml.RNode) error {
			result = nodes
			return nil
		})},
	}

	if err := pp.Execute(); err != nil {
		return nil, err
	}

	return result, nil
}

// Read allows this pipeline to become an input to a subsequent pipeline.
func (p *Pipeline) Read() ([]*yaml.RNode, error) {
	return p.Execute()
}
