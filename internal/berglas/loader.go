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
	"strings"

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	"sigs.k8s.io/kustomize/v3/k8sdeps/validator"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/types"
)

var _ ifc.Loader = &Loader{}

// Loader is a Kustomize Loader that actually just pulls secret data using the Berglas API.
type Loader struct {
	ctx       context.Context
	client    *berglas.Client
	validator ifc.Validator
}

func NewLoader(ctx context.Context) (*Loader, error) {
	c, err := berglas.New(ctx)
	if err != nil {
		return nil, err
	}
	return &Loader{
		ctx:       ctx,
		client:    c,
		validator: validator.NewKustValidator(),
	}, nil
}

func (l *Loader) Load(location string) ([]byte, error) {
	if r, err := berglas.ParseReference(l.Root() + location); err != nil {
		return nil, err
	} else {
		// TODO Where do we get the generation from?
		return l.client.Access(l.ctx, &berglas.AccessRequest{Bucket: r.Bucket(), Object: r.Object()})
	}
}

func (l *Loader) LoadKvPairs(args types.GeneratorArgs) ([]types.Pair, error) {
	var pairs []types.Pair

	// We don't support env files sources
	if len(args.EnvSources) > 0 || args.EnvSource != "" {
		return nil, fmt.Errorf("env sources are not supported")
	}

	// Literal sources are simple enough
	for _, ls := range args.LiteralSources {
		if strings.Index(ls, "=") <= 0 {
			return nil, fmt.Errorf("invalid literal: %s", ls)
		}
		p := strings.SplitN(ls, "=", 2)
		pairs = append(pairs, types.Pair{Key: p[0], Value: strings.Trim(p[1], `"'`)})
	}

	// File sources call Load to actually pull in the secret data
	for _, fs := range args.FileSources {
		var key, location string
		switch strings.Count(fs, "=") {
		case 0:
			key = path.Base(fs)
			location = fs
		case 1:
			p := strings.SplitN(fs, "=", 2)
			key = p[0]
			location = p[1]
		default:
			return nil, fmt.Errorf("invalid file: %s", fs)
		}

		if v, err := l.Load(location); err != nil {
			return nil, err
		} else {
			pairs = append(pairs, types.Pair{Key: key, Value: string(v)})
		}
	}

	return pairs, nil
}

func (l *Loader) Validator() ifc.Validator {
	return l.validator
}

func (*Loader) Root() string {
	return "berglas://"
}

func (*Loader) New(newRoot string) (ifc.Loader, error) {
	return nil, fmt.Errorf("changing roots is not supported")
}

func (*Loader) Cleanup() error {
	return nil
}
