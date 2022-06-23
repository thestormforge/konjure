package pipes

import "sigs.k8s.io/kustomize/kyaml/yaml"

// ReaderFunc is an adapter to allow the use of ordinary functions as a kio.Reader.
type ReaderFunc func() ([]*yaml.RNode, error)

// Read evaluates the typed function.
func (f ReaderFunc) Read() ([]*yaml.RNode, error) { return f() }
