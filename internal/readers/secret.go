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

package readers

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/sethvargo/go-password/password"
	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type SecretReader struct {
	konjurev1beta2.Secret
}

func (r *SecretReader) Read() ([]*yaml.RNode, error) {
	// Build the basic secret node
	n, err := yaml.FromMap(map[string]interface{}{"apiVersion": "v1", "kind": "Secret"})
	if err != nil {
		return nil, err
	}
	if err := n.PipeE(yaml.SetK8sName(r.SecretName)); err != nil {
		return nil, err
	}
	if r.Type != "" {
		if err := n.PipeE(yaml.SetField("type", yaml.NewStringRNode(r.Type))); err != nil {
			return nil, err
		}
	}

	// Add all the secret data
	if err := n.PipeE(yaml.Tee(
		yaml.FilterFunc(r.literals),
		yaml.FilterFunc(r.files),
		yaml.FilterFunc(r.envs),
		yaml.FilterFunc(r.uuids),
		yaml.FilterFunc(r.ulids),
		yaml.FilterFunc(r.passwords),
	)); err != nil {
		return nil, err
	}

	return []*yaml.RNode{n}, nil
}

func (r *SecretReader) literals(n *yaml.RNode) (*yaml.RNode, error) {
	if len(r.LiteralSources) == 0 {
		return n, nil
	}

	m := make(map[string]string)
	for _, s := range r.LiteralSources {
		items := strings.SplitN(s, "=", 2)
		if items[0] == "" || len(items) != 2 {
			return nil, fmt.Errorf("invalid literal, expected key=value: %s", s)
		}
		m[items[0]] = strings.Trim(items[1], `"'`)
	}

	return n, n.LoadMapIntoSecretData(m)
}

func (r *SecretReader) files(n *yaml.RNode) (*yaml.RNode, error) {
	if len(r.FileSources) == 0 {
		return n, nil
	}

	m := make(map[string]string)
	for _, s := range r.FileSources {
		items := strings.SplitN(s, "=", 3)
		switch len(items) {
		case 1:
			data, err := os.ReadFile(items[0])
			if err != nil {
				return nil, err
			}
			m[path.Base(items[0])] = string(data)

		case 2:
			if items[0] == "" || items[1] == "" {
				return nil, fmt.Errorf("key or file path is missing: %s", s)
			}

			data, err := os.ReadFile(items[1])
			if err != nil {
				return nil, err
			}
			m[items[0]] = string(data)

		default:
			return nil, fmt.Errorf("key names or file paths cannot contain '='")
		}
	}

	return n, n.LoadMapIntoSecretData(m)
}

func (r *SecretReader) envs(n *yaml.RNode) (*yaml.RNode, error) {
	if len(r.EnvSources) == 0 {
		return n, nil
	}

	m := make(map[string]string)
	for _, s := range r.EnvSources {
		data, err := os.ReadFile(s)
		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(bytes.NewReader(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})))
		currentLine := 0
		for scanner.Scan() {
			currentLine++

			line := scanner.Bytes()
			if !utf8.Valid(line) {
				return nil, fmt.Errorf("line %d has invalid UTF-8 bytes: %s", currentLine, string(line))
			}

			line = bytes.TrimLeftFunc(line, unicode.IsSpace)
			if len(line) == 0 || line[0] == '#' {
				continue
			}

			items := strings.SplitN(string(line), "=", 2)
			if len(items) == 2 {
				m[items[0]] = items[1]
			} else {
				m[items[0]] = os.Getenv(items[0])
			}
		}
	}

	return n, n.LoadMapIntoSecretData(m)
}

func (r *SecretReader) uuids(n *yaml.RNode) (*yaml.RNode, error) {
	if len(r.UUIDSources) == 0 {
		return n, nil
	}

	m := make(map[string]string)
	for _, s := range r.UUIDSources {
		v, err := uuid.NewRandom()
		if err != nil {
			return nil, err
		}
		m[s] = v.String()
	}

	return n, n.LoadMapIntoSecretData(m)
}

func (r *SecretReader) ulids(n *yaml.RNode) (*yaml.RNode, error) {
	if len(r.ULIDSources) == 0 {
		return n, nil
	}

	m := make(map[string]string)
	for _, s := range r.ULIDSources {
		v, err := ulid.New(ulid.Now(), rand.Reader)
		if err != nil {
			return nil, err
		}
		m[s] = v.String()
	}

	return n, n.LoadMapIntoSecretData(m)
}

func (r *SecretReader) passwords(n *yaml.RNode) (*yaml.RNode, error) {
	if len(r.PasswordSources) == 0 {
		return n, nil
	}

	gen, err := password.NewGenerator(r.PasswordOptions)
	if err != nil {
		return nil, err
	}

	m := make(map[string]string)
	for i := range r.PasswordSources {
		pwd, err := gen.Generate(passwordArgs(&r.PasswordSources[i]))
		if err != nil {
			return nil, err
		}
		m[r.PasswordSources[i].Key] = pwd
	}

	return n, n.LoadMapIntoSecretData(m)
}

func passwordArgs(s *konjurev1beta2.PasswordRecipe) (length int, numDigits int, numSymbols int, noUpper bool, allowRepeat bool) {
	if s.Length != nil {
		length = *s.Length
	}
	if s.NumDigits != nil {
		numDigits = *s.NumDigits
	}
	if s.NumSymbols != nil {
		numSymbols = *s.NumSymbols
	}
	if s.NoUpper != nil {
		noUpper = *s.NoUpper
	}
	if s.AllowRepeat != nil {
		allowRepeat = *s.AllowRepeat
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

	return length, numDigits, numSymbols, noUpper, allowRepeat
}
