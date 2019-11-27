/*
Copyright 2019 GramLabs, Inc.

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

package berglas

import (
	"context"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	"github.com/google/go-jsonnet"
	"sigs.k8s.io/kustomize/api/ifc"
)

// AsFileSource returns a KV loader file source for a Berglas reference. Returns an error
// if the Berglas reference cannot be parsed.
func AsFileSource(s string) (string, error) {
	ref, err := berglas.ParseReference(s)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	key := ref.Filepath()
	if key == "" {
		key = ref.Object()
	}
	buf.WriteString(path.Base(key))
	buf.WriteByte('=')
	buf.WriteString(ref.Bucket())
	buf.WriteByte('/')
	buf.WriteString(ref.Object())
	if gen := ref.Generation(); gen != 0 {
		buf.WriteByte('#')
		buf.WriteString(strconv.FormatInt(gen, 10))
	}
	return buf.String(), nil
}

var _ ifc.Loader = &SecretImporter{}
var _ jsonnet.Importer = &SecretImporter{}

// SecretImporter is a Jsonnet importer and Kustomize loader backed by the Berglas API
type SecretImporter struct {
}

// NewLoader returns a SecretImporter as a loader
func NewLoader() ifc.Loader {
	return &SecretImporter{}
}

// Root returns the prefix required by the Berglas IsReference function
func (si *SecretImporter) Root() string { return "berglas://" }

// Cleanup does nothing
func (si *SecretImporter) Cleanup() error { return nil }

// New is not supported for this loader
func (si *SecretImporter) New(newRoot string) (ifc.Loader, error) {
	return nil, fmt.Errorf("cannot create new roots for Berglas")
}

// Load will access a Berglas secret using a reference combined from the root and supplied location
func (si *SecretImporter) Load(location string) ([]byte, error) {
	ref, err := berglas.ParseReference(si.Root() + location)
	if err != nil {
		return nil, err
	}

	return berglas.Access(context.TODO(), &berglas.AccessRequest{
		Bucket:     ref.Bucket(),
		Object:     ref.Object(),
		Generation: ref.Generation(),
	})
}

// Import will access a Berglas secret for use in a Jsonnet program
func (si *SecretImporter) Import(importedFrom, importedPath string) (jsonnet.Contents, string, error) {
	ref, err := berglas.ParseReference(importedPath)
	if err != nil {
		return jsonnet.Contents{}, "", err
	}

	b, err := berglas.Access(context.TODO(), &berglas.AccessRequest{
		Bucket:     ref.Bucket(),
		Object:     ref.Object(),
		Generation: ref.Generation(),
	})
	if err != nil {
		return jsonnet.Contents{}, "", err
	}
	return jsonnet.MakeContents(string(b)), importedPath, nil
}
