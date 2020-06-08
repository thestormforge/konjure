package berglas

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	kwhlog "github.com/slok/kubewebhook/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/api/kv"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
)

// Mutator performs Berglas mutations on pod templates
type Mutator struct {
	h       *resmap.PluginHelpers
	genOpts *types.GeneratorOptions
	secrets resmap.ResMap
	logger  kwhlog.Logger
}

// NewMutator returns a new Berglas mutator from the specified Kustomize helpers
func NewMutator(h *resmap.PluginHelpers, opts *types.GeneratorOptions) *Mutator {
	m := &Mutator{
		h:       h,
		genOpts: opts,
		logger:  kwhlog.Dummy,
	}

	if opts != nil {
		m.secrets = resmap.New()
	}

	return m
}

// FlushSecrets removes the secrets stored in the mutator and appends them to the supplied resource map
func (m *Mutator) FlushSecrets(rm resmap.ResMap) error {
	var err error
	if m.secrets != nil {
		err = rm.AppendAll(m.secrets)
		m.secrets.Clear()
	}
	return err
}

// Mutate will alter the pod template to account for Berglas references, returning true if changes were made
func (m *Mutator) Mutate(template *corev1.PodTemplateSpec) (bool, error) {
	if m.genOpts != nil {
		return m.mutateTemplateWithSecrets(template)
	}
	return m.mutate(context.TODO(), template)
}

// Mutation with secrets does the secret lookup now instead of in the container

func (m *Mutator) mutateTemplateWithSecrets(template *corev1.PodTemplateSpec) (bool, error) {
	mutated := false

	for i, c := range template.Spec.InitContainers {
		if c, didMutate, err := m.mutateContainerWithSecrets(&c); err != nil {
			return mutated, err
		} else if didMutate {
			mutated = true
			template.Spec.InitContainers[i] = *c
		}
	}

	for i, c := range template.Spec.Containers {
		if c, didMutate, err := m.mutateContainerWithSecrets(&c); err != nil {
			return mutated, err
		} else if didMutate {
			mutated = true
			template.Spec.Containers[i] = *c
		}
	}

	// TODO We need to create itemized secret volumes to rename object identifiers to destination file names
	for _, r := range m.secrets.Resources() {
		mutated = true
		template.Spec.Volumes = append(template.Spec.Volumes, corev1.Volume{
			Name: r.GetName(),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: r.GetName(),
				},
			},
		})
	}

	return mutated, nil
}

func (m *Mutator) mutateContainerWithSecrets(c *corev1.Container) (*corev1.Container, bool, error) {
	mutated := false
	for _, e := range c.Env {
		if !berglas.IsReference(e.Value) {
			continue
		}

		// Parse the environment variable value as Berglas reference
		r, err := parseReference(e.Value)
		if err != nil {
			return c, mutated, err
		}

		fs, err := AsFileSource(e.Value)
		if err != nil {
			return c, mutated, err
		}

		// Create a resource map with a secret that we can merge into the existing collection
		args := types.SecretArgs{}
		args.Name = r.Bucket()
		args.FileSources = []string{fs}
		args.Options = m.genOpts
		sm, err := m.h.ResmapFactory().FromSecretArgs(kv.NewLoader(NewLoader(), m.h.Validator()), args)
		if err != nil {
			return c, mutated, err
		}

		// Merge the generated secret into the existing collection
		err = m.secrets.AbsorbAll(sm)
		if err != nil {
			return c, mutated, err
		}

		// Replace the environment variable value with the path
		if r.Filepath() != "" {
			mutated = true
			e.Value = r.Filepath()
		} else {
			// Do not allow environment variables to contain sensitive information in the generated manifests
			// TODO Should this be an error? Or do we just silently ignore it like when len(command) == 0...
			continue
		}

		// Add a mount to get the secret where it was requested
		// TODO If this volume mount is using the secret name, we need to ensure we aren't adding it multiple times
		c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
			Name:      args.Name,
			ReadOnly:  true,
			MountPath: e.Value,            // TODO How should we be mounting secrets
			SubPath:   path.Base(e.Value), // TODO "
		})
	}

	return c, mutated, nil
}

func parseReference(s string) (*berglas.Reference, error) {
	if u, err := url.Parse(s); err == nil {
		q := u.Query()
		if d := q.Get("destination"); d == "tempfile" || d == "tmpfile" {
			// TODO This needs to be a viable non-conflicting random value within the context of this transformation
			s = strings.Replace(s, d, "/tmp/berglas-XXXXXX", 1)
		}
	}
	return berglas.ParseReference(s)
}

// The rest of this is the mutating webhook from https://github.com/GoogleCloudPlatform/berglas/blob/v0.4.0/examples/kubernetes/main.go
// This code was released under the Apache 2.0 license

const (
	// berglasContainer is the default berglas container from which to pull the
	// berglas binary.
	berglasContainer = "gcr.io/berglas/berglas:latest"

	// binVolumeName is the name of the volume where the berglas binary is stored.
	binVolumeName = "berglas-bin"

	// binVolumeMountPath is the mount path where the berglas binary can be found.
	binVolumeMountPath = "/berglas/bin/"
)

// binInitContainer is the container that pulls the berglas binary executable
// into a shared volume mount.
var binInitContainer = corev1.Container{
	Name:            "copy-berglas-bin",
	Image:           berglasContainer,
	ImagePullPolicy: corev1.PullIfNotPresent,
	Command: []string{"sh", "-c",
		fmt.Sprintf("cp /bin/berglas %s", binVolumeMountPath)},
	VolumeMounts: []corev1.VolumeMount{
		{
			Name:      binVolumeName,
			MountPath: binVolumeMountPath,
		},
	},
}

// binVolume is the shared, in-memory volume where the berglas binary lives.
var binVolume = corev1.Volume{
	Name: binVolumeName,
	VolumeSource: corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{
			Medium: corev1.StorageMediumMemory,
		},
	},
}

// binVolumeMount is the shared volume mount where the berglas binary lives.
var binVolumeMount = corev1.VolumeMount{
	Name:      binVolumeName,
	MountPath: binVolumeMountPath,
	ReadOnly:  true,
}

// Mutate implements MutateFunc and provides the top-level entrypoint for object
// mutation.
func (m *Mutator) mutate(ctx context.Context, pod *corev1.PodTemplateSpec) (bool, error) {
	m.logger.Infof("calling mutate")

	mutated := false

	for i, c := range pod.Spec.InitContainers {
		c, didMutate := m.mutateContainer(ctx, &c)
		if didMutate {
			mutated = true
			pod.Spec.InitContainers[i] = *c
		}
	}

	for i, c := range pod.Spec.Containers {
		c, didMutate := m.mutateContainer(ctx, &c)
		if didMutate {
			mutated = true
			pod.Spec.Containers[i] = *c
		}
	}

	// If any of the containers requested berglas secrets, mount the shared volume
	// and ensure the berglas binary is available via an init container.
	if mutated {
		pod.Spec.Volumes = append(pod.Spec.Volumes, binVolume)
		pod.Spec.InitContainers = append([]corev1.Container{binInitContainer},
			pod.Spec.InitContainers...)
	}

	return mutated, nil
}

// mutateContainer mutates the given container, updating the volume mounts and
// command if it contains berglas references.
func (m *Mutator) mutateContainer(_ context.Context, c *corev1.Container) (*corev1.Container, bool) {
	// Ignore if there are no berglas references in the container.
	if !m.hasBerglasReferences(c.Env) {
		return c, false
	}

	// Berglas prepends the command from the podspec. If there's no command in the
	// podspec, there's nothing to append. Note: this is the command in the
	// podspec, not a CMD or ENTRYPOINT in a Dockerfile.
	if len(c.Command) == 0 {
		m.logger.Warningf("cannot apply berglas to %s: container spec does not define a command", c.Name)
		return c, false
	}

	// Add the shared volume mount
	c.VolumeMounts = append(c.VolumeMounts, binVolumeMount)

	// Prepend the command with berglas exec --
	original := append(c.Command, c.Args...)
	c.Command = []string{binVolumeMountPath + "berglas"}
	c.Args = append([]string{"exec", "--"}, original...)

	return c, true
}

// hasBerglasReferences parses the environment and returns true if any of the
// environment variables includes a berglas reference.
func (m *Mutator) hasBerglasReferences(env []corev1.EnvVar) bool {
	for _, e := range env {
		if berglas.IsReference(e.Value) {
			return true
		}
	}
	return false
}
