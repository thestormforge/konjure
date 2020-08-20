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
	"fmt"

	"github.com/carbonrelay/konjure/internal/kustomize"
	"github.com/carbonrelay/konjure/internal/secrets"
	"github.com/spf13/cobra"
)

// NewSecretGeneratorExecPlugin creates a new command for generating enhanced secrets as an executable plugin
func NewSecretGeneratorExecPlugin() *cobra.Command {
	p := &plugin{}
	cmd := kustomize.NewPluginRunner(p, kustomize.WithConfigType("konjure.carbonrelay.com", "v1beta1", "SecretGenerator"))
	return cmd
}

// NewSecretGeneratorCommand creates a new command for generating enhanced secrets from the CLI
func NewSecretGeneratorCommand() *cobra.Command {
	p := &plugin{}
	f := &secretFlags{}
	cmd := kustomize.NewPluginRunner(p, f.withPreRun(p))
	cmd.Use = "secret"
	cmd.Short = "Generate secrets"
	cmd.Flags().StringVar(&p.Name, "name", "", "name of the secret to generate")
	cmd.Flags().StringToStringVar(&f.passPhrases, "passphrase", nil, "GPG pass phrases for encrypted keys, e.g. 'my_secret.json=env:SECRETKEY'")
	cmd.Flags().StringToStringVar(&f.literals, "literal", nil, "literal `name=value` pair")
	cmd.Flags().StringArrayVar(&p.FileSources, "file", nil, "file `path` to include")
	cmd.Flags().StringArrayVar(&p.EnvSources, "env", nil, "env `file` to read")
	cmd.Flags().StringToStringVar(&f.passwords, "password", nil, "password `spec` to generate, e.g. 'mypassword=length:5,numDigits:2'")
	cmd.Flags().StringArrayVar(&p.UUIDSources, "uuid", nil, "UUID `key` to generate")
	cmd.Flags().StringArrayVar(&p.ULIDSources, "ulid", nil, "ULID `key` to generate")
	cmd.Flags().StringToStringVar(&f.secretManagerSecrets, "secret-manager", nil, "GCP Secret Manager `ref`, e.g. 'mysecret=project/secret")
	return cmd
}

type secretFlags struct {
	passPhrases          map[string]string
	literals             map[string]string
	passwords            map[string]string
	secretManagerSecrets map[string]string
}

// withPreRun will apply the stored flags to a plugin instance
func (f *secretFlags) withPreRun(p *plugin) kustomize.RunnerOption {
	return kustomize.WithPreRunE(func(cmd *cobra.Command, args []string) error {
		p.PassPhrases = make(map[string]secrets.PassPhrase, len(f.passPhrases))
		for k, v := range f.passPhrases {
			p.PassPhrases[k] = secrets.PassPhrase(v)
		}
		for k, v := range f.literals {
			p.LiteralSources = append(p.LiteralSources, fmt.Sprintf("%s=%s", k, v))
		}
		for k, v := range f.passwords {
			s := PasswordRecipe{Key: k}
			s.Parse(v)
			p.PasswordSources = append(p.PasswordSources, s)
		}
		for k, v := range f.secretManagerSecrets {
			s := SecretManagerReference{Key: k}
			s.Parse(v)
			p.SecretManagerSources = append(p.SecretManagerSources, s)
		}
		return nil
	})
}
