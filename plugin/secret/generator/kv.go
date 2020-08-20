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
	"crypto/rand"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/oklog/ulid"
	"github.com/sethvargo/go-password/password"
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

func (r *PasswordRecipe) Parse(spec string) {
	for _, s := range strings.Split(spec, ",") {
		p := strings.SplitN(s, ":", 2)
		if len(p) == 2 {
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
	}
}

type SecretManagerReference struct {
	Key     string `json:"key,omitempty"`
	Project string `json:"project"`
	Secret  string `json:"secret"`
	Version string `json:"version,omitempty"`
}

func (r *SecretManagerReference) Parse(spec string) {
	parts := strings.SplitN(spec, "/", 3)
	r.Project = parts[0]
	if len(parts) > 1 {
		r.Secret = parts[1]
	}
	if len(parts) > 2 {
		r.Version = parts[2]
	}
}

func passwordsAsLiteralSources(options *password.GeneratorInput, recipes []PasswordRecipe) ([]string, error) {
	if len(recipes) == 0 {
		return nil, nil
	}

	gen, err := password.NewGenerator(options)
	if err != nil {
		return nil, err
	}

	sources := make([]string, len(recipes))
	for i := range recipes {
		pwd, err := recipes[i].Generate(gen)
		if err != nil {
			return nil, err
		}
		sources[i] = fmt.Sprintf("%s=%s", recipes[i].Key, pwd)
	}
	return sources, nil
}

func uuidsAsLiteralSources(uuids []string) ([]string, error) {
	if len(uuids) == 0 {
		return nil, nil
	}

	sources := make([]string, len(uuids))
	for i := range uuids {
		v, err := uuid.NewRandom()
		if err != nil {
			return nil, err
		}
		sources[i] = fmt.Sprintf("%s=%s", uuids[i], v.String())
	}
	return sources, nil
}

func ulidsAsLiteralSources(ulids []string) ([]string, error) {
	if len(ulids) == 0 {
		return nil, nil
	}

	sources := make([]string, len(ulids))
	for i := range ulids {
		v, err := ulid.New(ulid.Now(), rand.Reader)
		if err != nil {
			return nil, err
		}
		sources[i] = fmt.Sprintf("%s=%s", ulids[i], v.String())
	}
	return sources, nil
}

func secretManagerSecretsAsFileSources(refs []SecretManagerReference) ([]string, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	sources := make([]string, len(refs))
	for i := range refs {
		var buf strings.Builder
		if key := refs[i].Key; key != "" {
			buf.WriteString(key)
			buf.WriteByte('=')
		}
		buf.WriteString("sm://")
		buf.WriteString(refs[i].Project)
		buf.WriteByte('/')
		buf.WriteString(refs[i].Secret)
		if version := refs[i].Version; version != "" {
			buf.WriteByte('#')
			buf.WriteString(version)
		}
		sources[i] = buf.String()
	}
	return sources, nil
}
