/*
Copyright 2021 GramLabs, Inc.

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

package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thestormforge/konjure/internal/readers"
	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"github.com/thestormforge/konjure/pkg/konjure"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func NewSecretCommand() *cobra.Command {
	f := secretFlags{}

	cmd := &cobra.Command{
		Use:    "secret",
		Short:  "Generate secrets",
		PreRun: f.preRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			return kio.Pipeline{
				Inputs:  []kio.Reader{&readers.SecretReader{Secret: f.Secret}},
				Outputs: []kio.Writer{&konjure.Writer{Writer: cmd.OutOrStdout()}},
			}.Execute()
		},
	}

	cmd.Flags().StringVar(&f.Name, "name", "", "name of the secret to generate")
	cmd.Flags().StringToStringVar(&f.literals, "literal", nil, "literal `name=value` pair")
	cmd.Flags().StringArrayVar(&f.FileSources, "file", nil, "file `path` to include")
	cmd.Flags().StringArrayVar(&f.EnvSources, "env", nil, "env `file` to read")
	cmd.Flags().StringArrayVar(&f.UUIDSources, "uuid", nil, "UUID `key` to generate")
	cmd.Flags().StringArrayVar(&f.ULIDSources, "ulid", nil, "ULID `key` to generate")
	cmd.Flags().StringToStringVar(&f.passwords, "password", nil, "password `spec` to generate, e.g. 'mypassword=length:5,numDigits:2'")

	return cmd
}

type secretFlags struct {
	konjurev1beta2.Secret
	literals  map[string]string
	passwords map[string]string
}

func (f *secretFlags) preRun(*cobra.Command, []string) {
	for k, v := range f.literals {
		f.LiteralSources = append(f.LiteralSources, fmt.Sprintf("%s=%s", k, v))
	}

	for k, v := range f.passwords {
		r := konjurev1beta2.PasswordRecipe{Key: k}
		for _, s := range strings.Split(v, ",") {
			p := strings.SplitN(s, ":", 2)
			if len(p) != 2 {
				continue
			}

			switch p[0] {
			case "length":
				l, _ := strconv.Atoi(p[1])
				r.Length = &l
			case "numDigits":
				nd, _ := strconv.Atoi(p[1])
				r.NumDigits = &nd
			case "numSymbols":
				ns, _ := strconv.Atoi(p[1])
				r.NumSymbols = &ns
			case "noUpper":
				nu, _ := strconv.ParseBool(p[1])
				r.NoUpper = &nu
			case "allowRepeat":
				ar, _ := strconv.ParseBool(p[1])
				r.AllowRepeat = &ar
			}
		}

		f.PasswordSources = append(f.PasswordSources, r)
	}
}
