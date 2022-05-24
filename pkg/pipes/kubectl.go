package pipes

import (
	"context"
	"os/exec"

	"github.com/spf13/pflag"
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

// AddFlags adds binding for configuring the `kubectl` fields.
func (k *Kubectl) AddFlags(flagSet *pflag.FlagSet) {
	flagSet.StringVar(&k.KubeConfig, "kubeconfig", k.KubeConfig, "")
	flagSet.StringVar(&k.Context, "context", k.Context, "")
	flagSet.StringVarP(&k.Namespace, "namespace", "n", k.Namespace, "")
}

// command creates a new executable command with the configured global flags and
// the supplied arguments.
func (k *Kubectl) command(ctx context.Context, args ...string) *exec.Cmd {
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
		Cmd: k.command(ctx, args...),
	}
}

// Create returns a sink for creating resources via kubectl.
func (k *Kubectl) Create(ctx context.Context, dryRun string) kio.Writer {
	args := []string{"create", "--filename", "-"}
	if dryRun != "" {
		args = append(args, "--dry-run", dryRun)
	}

	return &ExecWriter{
		Cmd: k.command(ctx, args...),
	}
}
