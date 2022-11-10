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
	"time"

	"github.com/thestormforge/konjure/pkg/pipes/karg"
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
	// The length of time to wait before giving up on a single request.
	RequestTimeout time.Duration
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
	if k.RequestTimeout != 0 {
		globalArgs = append(globalArgs, "--request-timeout", k.RequestTimeout.String())
	}

	return exec.CommandContext(ctx, name, append(globalArgs, args...)...)
}

// Reader returns a kio.Reader for the specified kubectl arguments.
func (k *Kubectl) Reader(ctx context.Context, args ...string) *ExecReader {
	return &ExecReader{
		Cmd: k.Command(ctx, append(args, "--output=yaml")...),
	}
}

// Writer returns a kio.Writer for the specified kubectl arguments.
func (k *Kubectl) Writer(ctx context.Context, args ...string) *ExecWriter {
	return &ExecWriter{
		Cmd: k.Command(ctx, append(args, "--filename=-")...),
	}
}

// Get returns a source for getting resources via kubectl.
func (k *Kubectl) Get(ctx context.Context, opts ...karg.GetOption) *ExecReader {
	r := k.Reader(ctx, "get")
	karg.WithGetOptions(r.Cmd, opts...)
	return r
}

// Create returns a sink for creating resources via kubectl.
func (k *Kubectl) Create(ctx context.Context, opts ...karg.CreateOption) *ExecWriter {
	w := k.Writer(ctx, "create")
	karg.WithCreateOptions(w.Cmd, opts...)
	return w
}

// Apply returns a sink for applying resources via kubectl.
func (k *Kubectl) Apply(ctx context.Context, opts ...karg.ApplyOption) *ExecWriter {
	w := k.Writer(ctx, "apply")
	karg.WithApplyOptions(w.Cmd, opts...)
	return w
}

// Delete returns a sink for deleting resources via kubectl.
func (k *Kubectl) Delete(ctx context.Context, opts ...karg.DeleteOption) *ExecWriter {
	w := k.Writer(ctx, "delete")
	karg.WithDeleteOptions(w.Cmd, opts...)
	return w
}

// Patch returns a sink for patching resources via kubectl.
func (k *Kubectl) Patch(ctx context.Context, opts ...karg.PatchOption) *ExecWriter {
	w := k.Writer(ctx, "patch")
	karg.WithPatchOptions(w.Cmd, opts...)
	return w
}

// Wait returns a command to wait for conditions via kubectl.
func (k *Kubectl) Wait(ctx context.Context, opts ...karg.WaitOption) *exec.Cmd {
	cmd := k.Command(ctx, "wait")
	karg.WithWaitOptions(cmd, opts...)
	return cmd
}
