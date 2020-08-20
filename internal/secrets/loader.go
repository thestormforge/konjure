/*
Copyright 2020 GramLabs, Inc.

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

package secrets

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-jsonnet"
	"sigs.k8s.io/kustomize/api/ifc"
)

// Loader of secret data. The supplied URLs use custom schemes to determine which source to get data from.
type Loader interface {
	Load(ctx context.Context, u url.URL) ([]byte, error)
}

// NewLoader returns a new secret loader.
func NewLoader(ctx context.Context) (Loader, error) {
	var l allTheLoaders
	var err error

	l.sm, err = NewSecretManagerLoader(ctx)
	if err != nil {
		return nil, err
	}

	return &l, nil
}

// allTheLoaders is a mux on the URL scheme to all of the individual loaders.
type allTheLoaders struct {
	sm *SecretManagerLoader
}

func (l *allTheLoaders) Load(ctx context.Context, u url.URL) ([]byte, error) {
	switch strings.ToLower(u.Scheme) {
	case "sm":
		return l.sm.Load(ctx, u)
	}
	return nil, fmt.Errorf("unable to load reference %q", u.String())
}

// NewKustomizeLoader wraps the Kustomize file loader to also consider secrets by URL.
func NewKustomizeLoader(files ifc.Loader, secrets Loader) ifc.Loader {
	return &kloader{files: files, secrets: secrets}
}

type kloader struct {
	files   ifc.Loader
	secrets Loader
}

func (k *kloader) Root() string {
	return k.files.Root()
}

func (k *kloader) New(newRoot string) (ifc.Loader, error) {
	r, err := k.files.New(newRoot)
	if err != nil {
		return nil, err
	}
	return &kloader{files: r, secrets: k.secrets}, nil
}

func (k *kloader) Load(location string) ([]byte, error) {
	u, err := parseURL(k.files.Root(), location)
	if err != nil {
		// TODO Should a URL parsing failure also delegate to the file loader?
		return nil, err
	}

	b, err := k.secrets.Load(context.TODO(), *u)
	if err != nil {
		return k.files.Load(location)
	}

	return b, nil
}

func (k *kloader) Cleanup() error {
	return k.files.Cleanup()
}

// NewJsonnetImporter wraps the Jsonnet file importer to also consider secrets by URL.
func NewJsonnetImporter(files jsonnet.Importer, secrets Loader) jsonnet.Importer {
	return &jimporter{files: files, secrets: secrets}
}

type jimporter struct {
	files   jsonnet.Importer
	secrets Loader
}

func (j *jimporter) Import(importedFrom, importedPath string) (contents jsonnet.Contents, foundAt string, err error) {
	u, err := parseURL(importedFrom, importedPath)
	if err != nil {
		return j.files.Import(importedFrom, importedPath)
	}

	b, err := j.secrets.Load(context.TODO(), *u)
	if err != nil {
		return j.files.Import(importedFrom, importedPath)
	}

	return jsonnet.MakeContents(string(b)), u.String(), nil
}

func parseURL(base, ref string) (*url.URL, error) {
	baseURL, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	return baseURL.Parse(ref)
}

func split(s string, sep byte) (string, string) {
	i := strings.IndexByte(s, sep)
	if i < 0 {
		return s, ""
	}
	return s[:i], s[i+1:]
}
