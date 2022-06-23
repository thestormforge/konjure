package pipes

import (
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ReaderFunc is an adapter to allow the use of ordinary functions as a kio.Reader.
type ReaderFunc func() ([]*yaml.RNode, error)

// Read evaluates the typed function.
func (r ReaderFunc) Read() ([]*yaml.RNode, error) { return r() }

// ErrorReader is an adapter to allow the use of an error as a kio.Reader.
type ErrorReader struct{ error }

// Reader returns the wrapped failure.
func (r ErrorReader) Read() ([]*yaml.RNode, error) { return nil, r }
