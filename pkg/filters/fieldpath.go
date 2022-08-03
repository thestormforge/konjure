/*
Copyright 2022 GramLabs, Inc.

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

package filters

import (
	"strings"
	"text/template"

	"sigs.k8s.io/kustomize/kyaml/utils"
)

// FieldPath evaluates a path template using the supplied context and then
// splits it into individual path segments (honoring escaped delimiters).
func FieldPath(p string, data map[string]string) ([]string, error) {
	// Evaluate the path as a Go Template
	t, err := template.New("path").
		Delims("{", "}").
		Option("missingkey=zero").
		Parse(p)
	if err != nil {
		return nil, err
	}

	var pathBuf strings.Builder
	if err := t.Execute(&pathBuf, data); err != nil {
		return nil, err
	}

	fieldPath, err := utils.SmarterPathSplitter(pathBuf.String(), "/"), nil
	if err != nil {
		return nil, err
	}

	for {
		switch {
		case len(fieldPath) == 0:
			return nil, nil
		case fieldPath[0] == "":
			fieldPath = fieldPath[1:]
		default:
			return fieldPath, nil
		}
	}
}
