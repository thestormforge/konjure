package konjure

import (
	"github.com/thestormforge/konjure/internal/readers"
	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

// ReaderFor returns a resource node reader or nil if the input is not recognized.
func ReaderFor(obj interface{}) kio.Reader {
	switch r := obj.(type) {
	case *konjurev1beta2.Resource:
		return &readers.ResourceReader{Resources: r.Resources}
	case *konjurev1beta2.Helm:
		return readers.NewHelmReader(r)
	case *konjurev1beta2.Jsonnet:
		return readers.NewJsonnetReader(r)
	case *konjurev1beta2.Kubernetes:
		return readers.NewKubernetesReader(r)
	case *konjurev1beta2.Kustomize:
		return readers.NewKustomizeReader(r)
	case *konjurev1beta2.Secret:
		return &readers.SecretReader{Secret: *r}
	case *konjurev1beta2.Git:
		return &readers.GitReader{Git: *r}
	case *konjurev1beta2.HTTP:
		return &readers.HTTPReader{HTTP: *r}
	case *konjurev1beta2.File:
		return &readers.FileReader{File: *r}
	default:
		return nil
	}
}
