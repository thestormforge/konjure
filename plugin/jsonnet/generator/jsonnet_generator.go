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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/carbonrelay/konjure/internal/berglas"
	"github.com/carbonrelay/konjure/internal/secrets"
	"github.com/fatih/color"
	"github.com/google/go-jsonnet"
	"github.com/jsonnet-bundler/jsonnet-bundler/pkg"
	"github.com/jsonnet-bundler/jsonnet-bundler/pkg/jsonnetfile"
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
	l  secrets.Loader
	bi *berglas.SecretImporter

	Filename          string      `json:"filename"`
	Code              string      `json:"exec"`
	JsonnetPath       []string    `json:"jpath"`
	ExternalVariables []Parameter `json:"extVar"`
	TopLevelArguments []Parameter `json:"topLevelArg"`

	JsonnetBundlerPackageHome string `json:"jbPkgHome"`
	JsonnetBundlerRefresh     bool   `json:"jbRefresh"`
}

//noinspection GoUnusedGlobalVariable
var KustomizePlugin plugin

func (p *plugin) Config(h *resmap.PluginHelpers, c []byte) error {
	p.h = h
	p.fi = &jsonnet.FileImporter{}
	p.bi = &berglas.SecretImporter{}
	l, err := secrets.NewLoader(context.Background())
	if err != nil {
		return err
	}
	p.l = l
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Generate() (resmap.ResMap, error) {
	filename, input, err := p.readInput()
	if err != nil {
		return nil, err
	}

	if err := p.evalJsonnetBundler(); err != nil {
		return nil, err
	}

	p.evalJpath()

	vm := jsonnet.MakeVM()
	vm.Importer(p) // TODO vm.Importer(secrets.NewJsonnetImporter(p.fi, p.l)
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

// Import resolves Jsonnet import statements using the Berglas and the file system
func (p *plugin) Import(importedFrom, importedPath string) (jsonnet.Contents, string, error) {
	// Ignore errors from the Berglas SecretImporter and just fall back to the FileImporter
	if c, fp, err := p.bi.Import(importedFrom, importedPath); err == nil {
		return c, fp, nil
	}
	return secrets.NewJsonnetImporter(p.fi, p.l).Import(importedFrom, importedPath)
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

func (p *plugin) evalJsonnetBundler() error {
	// Attempt to find and load the Jsonnet Bundler file
	jbfilebytes, err := p.h.Loader().Load(jsonnetfile.File)
	if err != nil {
		return nil // Ignore errors, just return if the file isn't present
	}
	jsonnetFile, err := jsonnetfile.Unmarshal(jbfilebytes)
	if err != nil {
		return err
	}

	// Attempt to find and load the Jsonnet Bundler lock file
	jblockfilebytes, err := p.h.Loader().Load(jsonnetfile.LockFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	lockFile, err := jsonnetfile.Unmarshal(jblockfilebytes)
	if err != nil {
		return err
	}

	// Default the name of the package home directory
	if p.JsonnetBundlerPackageHome == "" {
		p.JsonnetBundlerPackageHome = "vendor"
	}

	// Only run if the package home directory is missing or refresh is enabled
	jsonnetHome := filepath.Join(p.h.Loader().Root(), p.JsonnetBundlerPackageHome)
	if !p.JsonnetBundlerRefresh {
		if _, err := os.Stat(jsonnetHome); err == nil { // No error from stat, something is there
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Join(jsonnetHome, ".tmp"), os.ModePerm); err != nil {
		return err
	}

	// Ignore output when ensuring dependencies are updated
	color.Output = ioutil.Discard
	_, err = pkg.Ensure(jsonnetFile, p.JsonnetBundlerPackageHome, lockFile.Dependencies)
	return err
}

func (p *plugin) evalJpath() {
	// Include the environment variable
	jsonnetPath := filepath.SplitList(os.Getenv("JSONNET_PATH"))
	for i := len(jsonnetPath) - 1; i >= 0; i-- {
		p.fi.JPaths = append(p.fi.JPaths, jsonnetPath[i])
	}

	// Include the jsonnet-bundler integration
	if p.JsonnetBundlerPackageHome != "" {
		p.fi.JPaths = append(p.fi.JPaths, p.JsonnetBundlerPackageHome)
	}

	// Include the configured paths
	p.fi.JPaths = append(p.fi.JPaths, p.JsonnetPath...)
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
