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

type ExecReader struct {
	Name string
	Args []string
	Env  map[string]string
}

func (r *ExecReader) Read() ([]*yaml.RNode, error) {
	cmd := exec.Command(r.Name, r.Args...)
	for k, v := range r.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	out, err := cmd.Output()
	if eerr, ok := err.(*exec.ExitError); ok {
		msg := strings.TrimSpace(string(eerr.Stderr))
		msg = strings.TrimPrefix(msg, "Error: ")
		return nil, fmt.Errorf("%s %w: %s", r.Name, err, msg)
	} else if err != nil {
		return nil, err
	}

	return kio.FromBytes(out)
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

	err := kio.Pipeline{
		Inputs:                p.Inputs,
		Filters:               p.Filters,
		ContinueOnEmptyResult: p.ContinueOnEmptyResult,
		Outputs: []kio.Writer{kio.WriterFunc(func(nodes []*yaml.RNode) error {
			result = nodes
			return nil
		})},
	}.Execute()

	if err != nil {
		return nil, err
	}

	return result, nil
}

// Read allows this pipeline to become an input to a subsequent pipeline.
func (p *Pipeline) Read() ([]*yaml.RNode, error) {
	return p.Execute()
}
