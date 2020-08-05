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
	"strconv"
	"strings"

	"github.com/carbonrelay/konjure/internal/kustomize"
	"github.com/spf13/cobra"
)

// NewRandomGeneratorExecPlugin creates a new command for running random as an executable plugin
func NewRandomGeneratorExecPlugin() *cobra.Command {
	p := &plugin{}
	cmd := kustomize.NewPluginRunner(p, kustomize.WithConfigType("konjure.carbonrelay.com", "v1beta1", "RandomGenerator"))
	return cmd
}

// NewRandomGeneratorCommand creates a new command for running random from the CLI
func NewRandomGeneratorCommand() *cobra.Command {
	p := &plugin{}
	f := &randomFlags{}
	cmd := kustomize.NewPluginRunner(p, f.withPreRun(p))
	cmd.Use = "random"
	cmd.Short = "Generate random secrets"
	cmd.Deprecated = "Random secret generation will be removed in the next release"
	cmd.Flags().StringVar(&p.Name, "name", "", "name of the secret to generate")
	cmd.Flags().StringToStringVarP(&f.passwords, "password", "P", nil, "password `spec` to generate, e.g. 'mypassword=length:5,numDigits:2'")
	cmd.Flags().StringArrayVarP(&p.UUIDSources, "uuid", "U", nil, "uuid `key` to generate")
	return cmd
}

type randomFlags struct {
	passwords map[string]string
}

// withPreRun will apply the stored flags to a plugin instance
func (f *randomFlags) withPreRun(p *plugin) kustomize.RunnerOption {
	return kustomize.WithPreRunE(func(cmd *cobra.Command, args []string) error {
		for k, v := range f.passwords {
			s := PasswordRecipe{Key: k}
			parsePasswordSpec(&s, v)
			p.PasswordSources = append(p.PasswordSources, s)
		}
		return nil
	})
}

func parsePasswordSpec(recipe *PasswordRecipe, spec string) {
	for _, r := range strings.Split(spec, ",") {
		p := strings.SplitN(r, ":", 2)
		if len(p) == 2 {
			switch p[0] {
			case "length":
				l, _ := strconv.Atoi(p[1])
				recipe.Length = &l
			case "numDigits":
				nd, _ := strconv.Atoi(p[1])
				recipe.NumDigits = &nd
			case "numSymbols":
				ns, _ := strconv.Atoi(p[1])
				recipe.NumSymbols = &ns
			case "noUpper":
				nu, _ := strconv.ParseBool(p[1])
				recipe.NoUpper = &nu
			case "allowRepeat":
				ar, _ := strconv.ParseBool(p[1])
				recipe.AllowRepeat = &ar
			}
		}
	}
}
