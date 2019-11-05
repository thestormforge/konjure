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

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	"github.com/google/go-jsonnet"
)

var _ jsonnet.Importer = &SecretImporter{}

// SecretImporter allows Berglas secrets to be accessed via import or importstr from Jsonnet
type SecretImporter struct {
	ctx    context.Context
	client *berglas.Client
}

// NewSecretImporter creates a new Berglas importer for Jsonnet
func NewSecretImporter(ctx context.Context) (*SecretImporter, error) {
	c, err := berglas.New(ctx)
	if err != nil {
		return nil, err
	}
	return &SecretImporter{
		ctx:    ctx,
		client: c,
	}, nil
}

func (importer *SecretImporter) Accept(importedFrom, importedPath string) bool {
	return berglas.IsReference(importedPath)
}

// Import fulfils the jsonnet.Importer contract
func (importer *SecretImporter) Import(importedFrom, importedPath string) (contents jsonnet.Contents, foundAt string, err error) {
	r, err := berglas.ParseReference(importedPath)
	if err != nil {
		return jsonnet.Contents{}, "", fmt.Errorf("couldn't open import %#v: not a valid Berglas reference", importedPath)
	}

	// TODO Where do we get the generation from?
	bytes, err := importer.client.Access(importer.ctx, &berglas.AccessRequest{Bucket: r.Bucket(), Object: r.Object()})
	if err != nil {
		return jsonnet.Contents{}, "", err
	}

	return jsonnet.MakeContents(string(bytes)), importedPath, nil
}
