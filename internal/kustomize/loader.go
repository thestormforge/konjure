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

package kustomize

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"

	"sigs.k8s.io/kustomize/v3/k8sdeps/validator"
	"sigs.k8s.io/kustomize/v3/pkg/fs"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	fLdr "sigs.k8s.io/kustomize/v3/pkg/loader"
	"sigs.k8s.io/kustomize/v3/pkg/types"
)

var _ ifc.Loader = &loader{}

type loader struct {
	ldr    ifc.Loader
	ctx    context.Context
	client *berglas.Client
}

// NewKonjureLoader creates a new resource loader for Konjure plugins. The returned loader is hybrid remote/local
// loader that can access resources from the specified target or from Berglas.
func NewKonjureLoader(ctx context.Context, target string) (ifc.Loader, error) {
	lr := fLdr.RestrictionRootOnly
	v := validator.NewKustValidator()
	fSys := fs.MakeFsOnDisk()

	ldr, err := fLdr.NewLoader(lr, v, filepath.Clean(target), fSys)
	if err != nil {
		return nil, err
	}

	c, err := berglas.New(ctx)
	if err != nil {
		return nil, err
	}

	return &loader{
		ldr:    ldr,
		ctx:    ctx,
		client: c,
	}, nil
}

// MustUseKonjureLoader will panic if the supplied loader is not a Konjure loader
func MustUseKonjureLoader(ldr ifc.Loader) ifc.Loader {
	if _, ok := ldr.(*loader); !ok {
		panic("must use Konjure loader")
	}
	return ldr
}

func (l *loader) Root() string {
	return l.ldr.Root()
}

func (l *loader) New(newRoot string) (ifc.Loader, error) {
	return l.ldr.New(newRoot)
}

func (l *loader) Cleanup() error {
	return l.ldr.Cleanup()
}

func (l *loader) Validator() ifc.Validator {
	return l.ldr.Validator()
}

func (l *loader) Load(location string) ([]byte, error) {
	if berglas.IsReference(location) {
		return l.loadBerglas(location)
	}
	return l.ldr.Load(location)
}

func (l *loader) LoadKvPairs(args types.GeneratorArgs) ([]types.Pair, error) {
	// Delegate to load all of the non-file sources
	noFilesArgs := args
	noFilesArgs.FileSources = nil
	kvs, err := l.ldr.LoadKvPairs(noFilesArgs)
	if err != nil {
		return nil, err
	}

	// Load the file sources passing through the enhanced "Load" method
	for _, s := range args.FileSources {
		k, loc, err := parseFileSource(s)
		if err != nil {
			return nil, err
		}
		content, err := l.Load(loc)
		if err != nil {
			return nil, err
		}
		kvs = append(kvs, types.Pair{Key: k, Value: string(content)})
	}

	return kvs, err
}

func (l *loader) loadBerglas(location string) ([]byte, error) {
	ref, err := berglas.ParseReference(location)
	if err != nil {
		return nil, err
	}

	return l.client.Access(l.ctx, &berglas.AccessRequest{
		Bucket:     ref.Bucket(),
		Object:     ref.Object(),
		Generation: ref.Generation(),
	})
}

// parseFileSource accepts "key=file" or "file" or "berglas://..."
func parseFileSource(source string) (string, string, error) {
	// Since the Berglas URL may have "=" in it (query parameters), handle that first
	if ref, err := berglas.ParseReference(source); err == nil {
		if ref.Filepath() != "" {
			return path.Base(ref.Filepath()), source, nil
		}
		return ref.Object(), source, nil
	}

	// You cannot use keys or files with "=" in their name or the syntax is ambigous
	n := strings.Count(source, "=")
	if n == 0 {
		return path.Base(source), source, nil
	}
	if n > 1 || (n == 1 && strings.Trim(source, "=") != source) {
		return "", "", fmt.Errorf("invalid file source: %s", source)
	}
	parts := strings.Split(source, "=")
	return parts[0], parts[1], nil
}
