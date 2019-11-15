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

package loader

import (
	"context"
	"path/filepath"

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/ifc"
	fLdr "sigs.k8s.io/kustomize/api/loader"
)

var _ ifc.Loader = &loader{}

type loader struct {
	ldr    ifc.Loader
	ctx    context.Context
	client *berglas.Client
}

// NewLoader creates a new resource loader for Konjure plugins. The returned loader is hybrid remote/local
// loader that can access resources from the specified target or from Berglas.
func NewLoader(ctx context.Context, target string) (ifc.Loader, error) {
	lr := fLdr.RestrictionRootOnly
	fSys := filesys.MakeFsOnDisk()

	ldr, err := fLdr.NewLoader(lr, filepath.Clean(target), fSys)
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

// WrapLoader ensures the supplied loader is of the correct type
func WrapLoader(ldr ifc.Loader, ctx context.Context) ifc.Loader {
	if _, ok := ldr.(*loader); ok {
		// TODO Don't ignore the context...
		return ldr
	}

	c, _ := berglas.New(ctx)
	return &loader{
		ldr:    ldr,
		ctx:    ctx,
		client: c,
	}
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

func (l *loader) Load(location string) ([]byte, error) {
	if l.client != nil && berglas.IsReference(location) {
		return l.loadBerglas(location)
	}
	return l.ldr.Load(location)
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
