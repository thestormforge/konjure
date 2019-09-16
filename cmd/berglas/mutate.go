package berglas

import (
	"fmt"
	"path"

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/types"
)

type BerglasMutator struct {
	mutateContainer func(*corev1.Container) (*corev1.Container, bool)
	mutatePodSpec   func(*corev1.PodSpec)

	client        *berglas.Client
	resMapFactory *resmap.Factory
	loader        ifc.Loader
	secrets       resmap.ResMap
	lastErr       error
}

func NewBerglasMutator(c *berglas.Client, f *resmap.Factory, l ifc.Loader) *BerglasMutator {
	m := &BerglasMutator{
		client:        c,
		resMapFactory: f,
		loader:        l,
		secrets:       resmap.New(),
	}

	if m.client != nil {
		m.mutateContainer = m.mutateContainerEnvironment
		m.mutatePodSpec = m.mutatePodSpecEnvironment
	} else {
		m.mutateContainer = m.mutateContainerCommand
		m.mutatePodSpec = m.mutatePodSpecCommand
	}

	return m
}

// Include secrets at build time
func (m *BerglasMutator) mutateContainerEnvironment(c *corev1.Container) (*corev1.Container, bool) {
	// Check for a failure from an earlier container
	if m.lastErr != nil {
		return c, false
	}

	mutated := false
	for _, e := range c.Env {
		if berglas.IsReference(e.Value) {
			// Parse the environment variable value as Berglas reference
			r, err := berglas.ParseReference(e.Value)
			if err != nil {
				m.lastErr = err
				return c, mutated
			}

			// Do not allow environment variables specifications to contain sensitive information
			if r.Filepath() == "" {
				m.lastErr = fmt.Errorf("direct environment variable replacement is not allowed for: %s", e.Name)
				return c, mutated
			}

			// Add a secret by creating a resource map that we can merge into the existing collection
			args := types.SecretArgs{}
			args.Name = r.Bucket()
			args.FileSources = append(args.FileSources, fmt.Sprintf("%s=%s/%s", r.Filepath(), r.Bucket(), r.Object()))
			sm, err := m.resMapFactory.FromSecretArgs(m.loader, nil, args)
			if err != nil {
				m.lastErr = err
				return c, mutated
			}
			err = m.secrets.AbsorbAll(sm)
			if err != nil {
				m.lastErr = err
				return c, mutated
			}

			// Add a mount to get the secret where it was requested
			// TODO There is going to be a problem with "tempfile" since the OS the build runs on may have a different TMP_DIR convention
			c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
				Name:      r.Bucket(),
				MountPath: r.Filepath(),
				SubPath:   path.Base(r.Filepath()),
				ReadOnly:  true,
			})

			// Replace the environment variable value with the path
			e.Value = r.Filepath()
			mutated = true
		}
	}

	return c, mutated
}

func (m *BerglasMutator) mutatePodSpecEnvironment(spec *corev1.PodSpec) {
	for _, r := range m.secrets.Resources() {
		spec.Volumes = append(spec.Volumes, corev1.Volume{
			Name: r.GetName(),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: r.GetName(),
				},
			},
		})
	}
}

// The rest of this is the mutating webhook from https://github.com/GoogleCloudPlatform/berglas/tree/master/examples/kubernetes

const (
	berglasContainer   = "gcr.io/berglas/berglas:latest"
	binVolumeName      = "berglas-bin"
	binVolumeMountPath = "/berglas/bin/"
)

var binInitContainer = corev1.Container{
	Name:            "copy-berglas-bin",
	Image:           berglasContainer,
	ImagePullPolicy: corev1.PullIfNotPresent,
	Command:         []string{"sh", "-c", fmt.Sprintf("cp /bin/berglas %s", binVolumeMountPath)},
	VolumeMounts: []corev1.VolumeMount{
		{
			Name:      binVolumeName,
			MountPath: binVolumeMountPath,
		},
	},
}

var binVolume = corev1.Volume{
	Name: binVolumeName,
	VolumeSource: corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{
			Medium: corev1.StorageMediumMemory,
		},
	},
}

var binVolumeMount = corev1.VolumeMount{
	Name:      binVolumeName,
	MountPath: binVolumeMountPath,
	ReadOnly:  true,
}

func (m *BerglasMutator) mutateTemplate(template *corev1.PodTemplateSpec) bool {
	mutated := false

	for i, c := range template.Spec.InitContainers {
		c, didMutate := m.mutateContainer(&c)
		if didMutate {
			mutated = true
			template.Spec.InitContainers[i] = *c
		}
	}

	for i, c := range template.Spec.Containers {
		c, didMutate := m.mutateContainer(&c)
		if didMutate {
			mutated = true
			template.Spec.Containers[i] = *c
		}
	}

	if mutated {
		m.mutatePodSpec(&template.Spec)
	}

	return mutated
}

func (m *BerglasMutator) mutatePodSpecCommand(spec *corev1.PodSpec) {
	spec.Volumes = append(spec.Volumes, binVolume)
	spec.InitContainers = append([]corev1.Container{binInitContainer}, spec.InitContainers...)
}

func (m *BerglasMutator) mutateContainerCommand(c *corev1.Container) (*corev1.Container, bool) {
	if !m.hasBerglasReferences(c.Env) {
		return c, false
	}
	if len(c.Command) == 0 {
		return c, false
	}
	c.VolumeMounts = append(c.VolumeMounts, binVolumeMount)
	original := append(c.Command, c.Args...)
	c.Command = []string{binVolumeMountPath + "berglas"}
	c.Args = append([]string{"exec", "--local", "--"}, original...)
	return c, true
}

func (m *BerglasMutator) hasBerglasReferences(env []corev1.EnvVar) bool {
	for _, e := range env {
		if berglas.IsReference(e.Value) {
			return true
		}
	}
	return false
}
