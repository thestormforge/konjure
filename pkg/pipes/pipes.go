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
