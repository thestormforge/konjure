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
	"fmt"

	"github.com/sethvargo/go-password/password"
	"sigs.k8s.io/kustomize/api/kv"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/pseudo/k8s/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/yaml"
)

// PasswordRecipe controls how passwords are generated
type PasswordRecipe struct {
	Key         string `json:"key"`
	Length      *int   `json:"length,omitempty"`
	NumDigits   *int   `json:"numDigits,omitempty"`
	NumSymbols  *int   `json:"numSymbols,omitempty"`
	NoUpper     *bool  `json:"noUpper,omitempty"`
	AllowRepeat *bool  `json:"allowRepeat,omitempty"`
}

type plugin struct {
	h *resmap.PluginHelpers

	GeneratorOptions *types.GeneratorOptions  `json:"generatorOptions,omitempty"`
	GeneratorInput   *password.GeneratorInput `json:"passwordOptions,omitempty"`
	Namespace        string                   `json:"namespace,omitempty"`
	Name             string                   `json:"name"`
	PasswordSources  []PasswordRecipe         `json:"passwords,omitempty"`
	UUIDSources      []string                 `json:"uuids,omitempty"`
}

//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Generate() (resmap.ResMap, error) {
	gen, err := password.NewGenerator(p.GeneratorInput)
	if err != nil {
		return nil, err
	}

	args := types.SecretArgs{}
	args.Namespace = p.Namespace
	args.Name = p.Name

	for _, s := range p.PasswordSources {
		pwd, err := s.Generate(gen)
		if err != nil {
			return nil, err
		}
		args.LiteralSources = append(args.LiteralSources, fmt.Sprintf("%s=%s", s.Key, pwd))
	}

	for _, s := range p.UUIDSources {
		args.LiteralSources = append(args.LiteralSources, fmt.Sprintf("%s=%s", s, uuid.NewUUID()))
	}

	return p.h.ResmapFactory().FromSecretArgs(
		kv.NewLoader(p.h.Loader(), p.h.Validator()),
		p.GeneratorOptions, args)
}

// Generate returns the password produced by the supplied generator using this recipe.
func (r *PasswordRecipe) Generate(gen password.PasswordGenerator) (string, error) {
	var length, numDigits, numSymbols int
	var noUpper, allowRepeat bool

	if r.Length != nil {
		length = *r.Length
	}
	if r.NumDigits != nil {
		numDigits = *r.NumDigits
	}
	if r.NumSymbols != nil {
		numSymbols = *r.NumSymbols
	}
	if r.NoUpper != nil {
		noUpper = *r.NoUpper
	}
	if r.AllowRepeat != nil {
		allowRepeat = *r.AllowRepeat
	}

	// TODO Is this reasonable default logic?
	if length == 0 {
		length = 64
	}
	if numDigits == 0 && numSymbols+10 <= length {
		numDigits = 10
	}
	if numSymbols == 0 && numDigits+10 <= length {
		numSymbols = 10
	}

	return gen.Generate(length, numDigits, numSymbols, noUpper, allowRepeat)
}
