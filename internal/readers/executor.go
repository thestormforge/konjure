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

type Cleaner interface {
	Clean() error
}

type CleanUpError []error

func (e CleanUpError) Error() string {
	var errStrings []string
	for _, err := range e {
		errStrings = append(errStrings, err.Error())
	}
	return strings.Join(errStrings, "\n")
}

type Cleaners []Cleaner

func (cs Cleaners) CleanUp() error {
	var errs CleanUpError
	for _, c := range cs {
		if err := c.Clean(); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

type Executor func(cmd *exec.Cmd) ([]byte, error)

func FromCommand(cmd *exec.Cmd, output Executor) ([]*yaml.RNode, error) {
	out, err := output(cmd)
	if eerr, ok := err.(*exec.ExitError); ok {
		msg := strings.TrimSpace(string(eerr.Stderr))
		msg = strings.TrimPrefix(msg, "Error: ")
		return nil, fmt.Errorf("%s %w: %s", filepath.Base(cmd.Path), err, msg)
	} else if err != nil {
		return nil, err
	}

	return kio.FromBytes(out)
}

var defaultExecutor = func(cmd *exec.Cmd) ([]byte, error) { return cmd.Output() }

type ExecReader exec.Cmd

func (cmd *ExecReader) Read() ([]*yaml.RNode, error) {
	return FromCommand((*exec.Cmd)(cmd), defaultExecutor)
}

type ErrorReader struct {
	err error
}

func (r *ErrorReader) Read() ([]*yaml.RNode, error) {
	return nil, r.err
}

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

type ExecutorMux struct {
	Git       Executor
	Helm      Executor
	Jsonnet   Executor
	Kubectl   Executor
	Kustomize Executor
}

func (e *ExecutorMux) HandleExecution(r kio.Reader) kio.Reader {
	// If it is a pipeline, recursively decorate the inputs
	if p, ok := r.(*Pipeline); ok {
		for i := range p.Inputs {
			p.Inputs[i] = e.HandleExecution(p.Inputs[i])
		}
		return p
	}

	// If it's not an exec reader, there is nothing we can do
	er, ok := r.(*ExecReader)
	if !ok {
		return r
	}

	// Check to see if we have an executor registered, if not just leave it alone
	exr := &executableReader{Command: (*exec.Cmd)(er)}
	switch filepath.Base(er.Path) {
	case "git":
		exr.Executor = e.Git
	case "helm":
		exr.Executor = e.Helm
	case "jsonnet":
		exr.Executor = e.Jsonnet
	case "kubectl":
		exr.Executor = e.Kubectl
	case "kustomize":
		exr.Executor = e.Kustomize
	}

	if exr.Executor != nil {
		return exr
	}
	return er
}

type executableReader struct {
	Command  *exec.Cmd
	Executor Executor
}

func (e *executableReader) Read() ([]*yaml.RNode, error) {
	return FromCommand(e.Command, e.Executor)
}
