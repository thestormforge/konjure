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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/google/go-jsonnet"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/yaml"
)

// Parameter defines either and external variable or top-level argument; except name, all are mutually exclusive.
type Parameter struct {
	Name       string `json:"name,omitempty"`
	String     string `json:"string,omitempty"`
	StringFile string `json:"stringFile,omitempty"`
	Code       string `json:"code,omitempty"`
	CodeFile   string `json:"codeFile,omitempty"`
}

type plugin struct {
	h  *resmap.PluginHelpers
	fi *jsonnet.FileImporter

	Filename          string      `json:"filename"`
	Code              string      `json:"exec"`
	JsonnetPath       []string    `json:"jpath"`
	ExternalVariables []Parameter `json:"extVar"`
	TopLevelArguments []Parameter `json:"topLevelArg"`
}

//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h
	p.fi = &jsonnet.FileImporter{}
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Generate() (resmap.ResMap, error) {
	filename, input, err := p.readInput()
	if err != nil {
		return nil, err
	}

	p.evalJpath()

	vm := jsonnet.MakeVM()
	vm.Importer(p)
	processParameters(p.ExternalVariables, vm.ExtVar, vm.ExtCode)
	processParameters(p.TopLevelArguments, vm.TLAVar, vm.TLACode)

	output, err := vm.EvaluateSnippet(filename, string(input))
	if err != nil {
		return nil, err
	}

	m, err := p.newResMapFromMultiDocumentJSON([]byte(output))
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Import resolves Jsonnet import statements using the Kustomize loader
func (p *plugin) Import(importedFrom, importedPath string) (jsonnet.Contents, string, error) {
	if b, err := p.h.Loader().Load(importedPath); err == nil {
		return jsonnet.MakeContents(string(b)), importedPath, nil
	}

	// Fallback to the standard Jsonnet implementation if there was an error
	return p.fi.Import(importedFrom, importedPath)
}

func (p *plugin) readInput() (string, []byte, error) {
	if p.Filename != "" {
		b, err := ioutil.ReadFile(p.Filename)
		return p.Filename, b, err
	}

	if p.Code != "" {
		return "<cmdline>", []byte(p.Code), nil
	}

	return "<empty>", nil, nil
}

func (p *plugin) evalJpath() {
	var evalJpath []string
	jsonnetPath := filepath.SplitList(os.Getenv("JSONNET_PATH"))
	for i := len(jsonnetPath) - 1; i >= 0; i-- {
		evalJpath = append(evalJpath, jsonnetPath[i])
	}
	p.fi.JPaths = append(evalJpath, p.JsonnetPath...)
}

func processParameters(params []Parameter, handleVar func(string, string), handleCode func(string, string)) {
	for _, p := range params {
		if p.String != "" {
			handleVar(p.Name, p.String)
		} else if p.StringFile != "" {
			handleCode(p.Name, fmt.Sprintf("importstr @'%s'", strings.ReplaceAll(p.StringFile, "'", "''")))
		} else if p.Code != "" {
			handleCode(p.Name, p.Code)
		} else if p.CodeFile != "" {
			handleCode(p.Name, fmt.Sprintf("import @'%s'", strings.ReplaceAll(p.StringFile, "'", "''")))
		}
	}
}

// newResMapFromMultiDocumentJSON inspects the supplied byte array to determine how it should be handled: if it
// is a JSON list, each item in the list is added to a new resource map; if the the command produces an object with a
// "kind" field, the contents are passed directly into the resource map; objects without a "kind" field are assumed
// to be a map of file names to document  contents and each field value is inserted to a new resource map honoring
// the order imposed by a sort of the keys.
func (p *plugin) newResMapFromMultiDocumentJSON(b []byte) (resmap.ResMap, error) {
	m := resmap.New()

	// This is JSON, we can trim the leading space
	j := bytes.TrimLeftFunc(b, unicode.IsSpace)
	if len(j) == 0 {
		return m, nil
	}

	rf := p.h.ResmapFactory().RF()

	if bytes.HasPrefix(j, []byte("[")) {
		// JSON list: just add each item as a new resource
		raw := make([]interface{}, 0)
		if err := json.Unmarshal(j, &raw); err != nil {
			return nil, err
		}
		for i := range raw {
			if o, ok := raw[i].(map[string]interface{}); ok {
				if err := m.Append(rf.FromMap(o)); err != nil {
					return nil, err
				}
			} else {
				return nil, fmt.Errorf("expected a list of objects")
			}
		}
		return m, nil
	}

	if bytes.HasPrefix(j, []byte("{")) {
		// JSON object: look for a "kind" field
		raw := make(map[string]interface{})
		if err := json.Unmarshal(j, &raw); err != nil {
			return nil, err
		}
		if _, ok := raw["kind"]; ok {
			// If there is a "kind" field, assume the factory will know what to do with it
			if err := m.Append(rf.FromMap(raw)); err != nil {
				return nil, err
			}
		} else {
			// Assume filename->object (where each object has a "kind"), preserve the order introduced by the filenames
			var filenames []string
			for k := range raw {
				filenames = append(filenames, k)
			}
			sort.Strings(filenames)

			for _, k := range filenames {
				if o, ok := raw[k].(map[string]interface{}); ok {
					if err := m.Append(rf.FromMap(o)); err != nil {
						return nil, err
					}
				} else {
					return nil, fmt.Errorf("expected a map of objects")
				}
			}
		}
		return m, nil
	}

	return nil, fmt.Errorf("expected JSON object or list")
}
