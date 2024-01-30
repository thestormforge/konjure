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
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// New returns a resource node reader or nil if the input is not recognized.
func New(res any) kio.Reader {
	// Construct a new reader based on the input type
	switch res := res.(type) {
	case *konjurev1beta2.Resource:
		return &ResourceReader{Resource: *res}
	case *konjurev1beta2.Helm:
		return &HelmReader{Helm: *res}
	case *konjurev1beta2.Jsonnet:
		return NewJsonnetReader(res)
	case *konjurev1beta2.Kubernetes:
		return &KubernetesReader{Kubernetes: *res}
	case *konjurev1beta2.Kustomize:
		return &KustomizeReader{Kustomize: *res}
	case *konjurev1beta2.Secret:
		return &SecretReader{Secret: *res}
	case *konjurev1beta2.Git:
		return &GitReader{Git: *res}
	case *konjurev1beta2.HTTP:
		return &HTTPReader{HTTP: *res}
	case *konjurev1beta2.File:
		return &FileReader{File: *res}
	}
	return nil
}

// Executor is function that returns the output of a command.
type Executor func(cmd *exec.Cmd) ([]byte, error)

// Runtime contains the base configuration for creating `exec.Cmd` instances.
type Runtime struct {
	// Bin can be configured to override the default path to the binary.
	Bin string
	// Executor can be set to change how the command is executed. If left `nil`,
	// commands will execute via their `Cmd.Output` function.
	Executor Executor
}

// command returns a new `exec.Cmd` runtime wrapper for the supplied command name.
func (rt *Runtime) command(defBin string) *command {
	bin := rt.Bin
	if bin == "" {
		bin = defBin
	}

	return &command{
		Cmd:      exec.Command(bin),
		Executor: rt.Executor,
	}
}

// command is a runtime wrapper for an `exec.Cmd`.
type command struct {
	*exec.Cmd
	Executor
}

// Output invokes the standard `Cmd.Output` function unless there is an explicit
// executor configured to handle execution.
func (cmd *command) Output() ([]byte, error) {
	if cmd.Executor != nil {
		return cmd.Executor(cmd.Cmd)
	}

	return cmd.Cmd.Output()
}

// Read allows the runtime command to act as a `kio.Reader` assuming the command
// emits YAML manifests to stdout.
func (cmd *command) Read() ([]*yaml.RNode, error) {
	out, err := cmd.Output()
	if err != nil {
		var eerr *exec.ExitError
		if errors.As(err, &eerr) {
			msg := strings.TrimSpace(string(eerr.Stderr))
			msg = strings.TrimPrefix(msg, "Error: ")
			return nil, fmt.Errorf("%s %w: %s", filepath.Base(cmd.Path), err, msg)
		}
		return nil, err
	}

	return kio.FromBytes(out)
}
