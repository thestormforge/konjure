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

package kv

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	kLdr "github.com/carbonrelay/konjure/internal/kustomize/loader"
	"sigs.k8s.io/kustomize/api/ifc"
	kkv "sigs.k8s.io/kustomize/api/kv"
	"sigs.k8s.io/kustomize/api/types"
)

var _ ifc.KvLoader = &loader{}

type loader struct {
	ldr ifc.Loader
	kv  ifc.KvLoader
}

func NewLoader(ldr ifc.Loader, validator ifc.Validator, ctx context.Context) ifc.KvLoader {
	return &loader{
		ldr: kLdr.WrapLoader(ldr, ctx),
		kv:  kkv.NewLoader(ldr, validator),
	}
}

func (l *loader) Validator() ifc.Validator {
	return l.kv.Validator()
}

func (l *loader) Load(args types.KvPairSources) ([]types.Pair, error) {
	// Delegate to load all of the non-file sources
	noFilesArgs := args
	noFilesArgs.FileSources = nil
	kvs, err := l.kv.Load(noFilesArgs)
	if err != nil {
		return nil, err
	}

	// Load the file sources passing through the enhanced parseFileSource method
	for _, s := range args.FileSources {
		k, loc, err := parseFileSource(s)
		if err != nil {
			return nil, err
		}
		content, err := l.ldr.Load(loc)
		if err != nil {
			return nil, err
		}
		kvs = append(kvs, types.Pair{Key: k, Value: string(content)})
	}

	return kvs, err
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
