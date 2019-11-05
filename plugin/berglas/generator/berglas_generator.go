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

package generator

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	berglas2 "github.com/carbonrelay/konjure/internal/berglas"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/types"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	ldr ifc.Loader
	rf  *resmap.Factory

	GeneratorOptions *types.GeneratorOptions `json:"generatorOptions,omitempty"`
	Namespace        string                  `json:"namespace,omitempty"`
	Name             string                  `json:"name"`
	References       []string                `json:"refs"`
}

var KustomizePlugin plugin

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) error {
	p.ldr = ldr
	p.rf = rf
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Generate() (resmap.ResMap, error) {
	// TODO Expose additional configuration options for the client
	bLdr, err := berglas2.NewLoader(context.Background())
	if err != nil {
		return nil, err
	}

	// Add a file source for each of the configured references
	args := types.SecretArgs{}
	args.Namespace = p.Namespace
	args.Name = p.Name
	for _, ref := range p.References {
		// TODO This drops the generation from the URI fragment
		r, err := berglas.ParseReference(ref)
		if err != nil {
			return nil, err
		}
		k := r.Filepath()
		if k != "" {
			k = path.Base(k)
		}
		fileSource := fmt.Sprintf("%s=%s/%s", k, r.Bucket(), r.Object())
		args.FileSources = append(args.FileSources, strings.TrimLeft(fileSource, "="))
	}

	// Generate the secret resource using the Berglas loader
	return p.rf.FromSecretArgs(bLdr, p.GeneratorOptions, args)
}
