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

package generator

import (
	"context"

	"github.com/carbonrelay/konjure/internal/secrets"
	"github.com/sethvargo/go-password/password"
	"sigs.k8s.io/kustomize/api/kv"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

// This plugin combines the standard secret generator with additional sources.

type plugin struct {
	h *resmap.PluginHelpers

	types.GeneratorArgs
	Type            string                   `json:"type,omitempty"`
	PasswordOptions *password.GeneratorInput `json:"passwordOptions,omitempty"`

	PasswordSources      []PasswordRecipe         `json:"passwords,omitempty"`
	UUIDSources          []string                 `json:"uuids,omitempty"`
	ULIDSources          []string                 `json:"ulids,omitempty"`
	SecretManagerSources []SecretManagerReference `json:"secretManagerSecrets,omitempty"`

	// TODO Random word combinations (moby, business)
	// TODO 1Password (via CLI)
	// TODO Vault
	// TODO AWS?
	// TODO GPG
}

//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(h *resmap.PluginHelpers, config []byte) error {
	p.h = h
	return yaml.Unmarshal(config, p)
}

func (p *plugin) Generate() (resmap.ResMap, error) {
	args := types.SecretArgs{
		GeneratorArgs: p.GeneratorArgs,
		Type:          p.Type,
	}

	// Now we add our custom sources as appropriate, filling them in as literals or files where necessary

	if literalSources, err := passwordsAsLiteralSources(p.PasswordOptions, p.PasswordSources); err != nil {
		return nil, err
	} else if literalSources != nil {
		args.LiteralSources = append(args.LiteralSources, literalSources...)
	}

	if literalSources, err := uuidsAsLiteralSources(p.UUIDSources); err != nil {
		return nil, err
	} else if literalSources != nil {
		args.LiteralSources = append(args.LiteralSources, literalSources...)
	}

	if literalSources, err := ulidsAsLiteralSources(p.ULIDSources); err != nil {
		return nil, err
	} else if literalSources != nil {
		args.LiteralSources = append(args.LiteralSources, literalSources...)
	}

	if fileSources, err := secretManagerSecretsAsFileSources(p.SecretManagerSources); err != nil {
		return nil, err
	} else if fileSources != nil {
		args.FileSources = append(args.FileSources, fileSources...)
	}

	// Determine what we need for a loader
	sldr, err := secrets.NewLoader(context.Background())
	if err != nil {
		return nil, err
	}
	ldr := secrets.NewKustomizeLoader(p.h.Loader(), sldr)

	// Generate the secret using the standard implementation
	return p.h.ResmapFactory().FromSecretArgs(kv.NewLoader(ldr, p.h.Validator()), args)
}
