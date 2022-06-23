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
	"context"
	"os/exec"

	"sigs.k8s.io/kustomize/kyaml/kio"
)

// Kubectl is used for executing `kubectl` as part of a KYAML pipeline.
type Kubectl struct {
	// The path the kubectl binary, defaults to `kubectl`.
	Bin string
	// The path to the kubeconfig.
	KubeConfig string
	// The context name.
	Context string
	// The namespace name.
	Namespace string
}

// Command creates a new executable command with the configured global flags and
// the supplied arguments.
func (k *Kubectl) Command(ctx context.Context, args ...string) *exec.Cmd {
	name := k.Bin
	if name == "" {
		name = "kubectl"
	}

	var globalArgs []string
	if k.KubeConfig != "" {
		globalArgs = append(globalArgs, "--kubeconfig", k.KubeConfig)
	}
	if k.Context != "" {
		globalArgs = append(globalArgs, "--context", k.Context)
	}
	if k.Namespace != "" {
		globalArgs = append(globalArgs, "--namespace", k.Namespace)
	}

	return exec.CommandContext(ctx, name, append(globalArgs, args...)...)
}

// Get returns a source for getting resources via kubectl.
func (k *Kubectl) Get(ctx context.Context, objs ...string) kio.Reader {
	args := []string{"get", "--output", "yaml"}
	args = append(args, objs...)

	return &ExecReader{
		Cmd: k.Command(ctx, args...),
	}
}

// Create returns a sink for creating resources via kubectl.
func (k *Kubectl) Create(ctx context.Context, dryRun string) kio.Writer {
	args := []string{"create", "--filename", "-"}
	if dryRun != "" {
		args = append(args, "--dry-run", dryRun)
	}

	return &ExecWriter{
		Cmd: k.Command(ctx, args...),
	}
}

// Apply returns a sink for applying resources via kubectl.
func (k *Kubectl) Apply(ctx context.Context, dryRun string) kio.Writer {
	args := []string{"apply", "--filename", "-"}
	if dryRun != "" {
		args = append(args, "--dry-run", dryRun)
	}

	return &ExecWriter{
		Cmd: k.Command(ctx, args...),
	}
}
