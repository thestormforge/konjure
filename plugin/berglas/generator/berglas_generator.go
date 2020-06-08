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
	"github.com/carbonrelay/konjure/internal/berglas"
	"sigs.k8s.io/kustomize/api/kv"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	h *resmap.PluginHelpers

	GeneratorOptions *types.GeneratorOptions `json:"generatorOptions,omitempty"`
	Namespace        string                  `json:"namespace,omitempty"`
	Name             string                  `json:"name"`
	References       []string                `json:"refs"`
}

//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Generate() (resmap.ResMap, error) {
	args := types.SecretArgs{}
	args.Namespace = p.Namespace
	args.Name = p.Name

	for _, ref := range p.References {
		fs, err := berglas.AsFileSource(ref)
		if err != nil {
			return nil, err
		}
		args.FileSources = append(args.FileSources, fs)
	}

	// Generate the ResMap
	args.Options = p.GeneratorOptions
	return p.h.ResmapFactory().FromSecretArgs(kv.NewLoader(berglas.NewLoader(), p.h.Validator()), args)
}
