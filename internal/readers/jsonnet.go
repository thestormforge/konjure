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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/google/go-jsonnet"
	jb "github.com/jsonnet-bundler/jsonnet-bundler/pkg"
	"github.com/jsonnet-bundler/jsonnet-bundler/pkg/jsonnetfile"
	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func NewJsonnetReader(js *konjurev1beta2.Jsonnet) kio.Reader {
	// Build the reader from the Jsonnet configuration
	r := &JsonnetReader{
		JsonnetBundlerPackageHome: js.JsonnetBundlerPackageHome,
		FileImporter: jsonnet.FileImporter{
			JPaths: js.JsonnetPath,
		},
		MakeVM: func() *jsonnet.VM {
			vm := jsonnet.MakeVM()
			processParameters(js.ExternalVariables, vm.ExtVar, vm.ExtCode)
			processParameters(js.TopLevelArguments, vm.TLAVar, vm.TLACode)
			return vm
		},
		Filename: js.Filename,
		Snippet:  js.Code,
	}

	// Setup Jsonnet Bundler
	if _, err := os.Stat(jsonnetfile.File); err != nil {
		r.JsonnetBundlerPackageHome = "" // No 'jsonnetfile.json'
	} else {
		if r.JsonnetBundlerPackageHome == "" {
			r.JsonnetBundlerPackageHome = "vendor"
		}
		r.FileImporter.JPaths = append(r.FileImporter.JPaths, js.JsonnetBundlerPackageHome)

		if _, err := os.Stat(r.JsonnetBundlerPackageHome); err == nil && !js.JsonnetBundlerRefresh {
			r.JsonnetBundlerPackageHome = "" // Vendor exists and should not be refreshed
		}
	}

	// Finish the path (it must be reversed since the file importer reads from the end)
	r.FileImporter.JPaths = append(r.FileImporter.JPaths, filepath.SplitList(os.Getenv("JSONNET_PATH"))...)
	for i, j := 0, len(r.FileImporter.JPaths)-1; i < j; i, j = i+1, j-1 {
		r.FileImporter.JPaths[i], r.FileImporter.JPaths[j] = r.FileImporter.JPaths[j], r.FileImporter.JPaths[i]
	}

	return r
}

type JsonnetReader struct {
	JsonnetBundlerPackageHome string
	FileImporter              jsonnet.FileImporter
	MakeVM                    func() *jsonnet.VM
	Filename                  string
	Snippet                   string
}

func (r *JsonnetReader) Read() ([]*yaml.RNode, error) {
	// Before we start, make sure the bundler is up-to-date (i.e. download `/vendor/`)
	if r.JsonnetBundlerPackageHome != "" {
		if err := r.bundlerEnsure(); err != nil {
			return nil, err
		}
	}

	// Get the configured VM factory or use the default
	makeVM := r.MakeVM
	if makeVM == nil {
		makeVM = jsonnet.MakeVM
	}

	// Create a new VM configured to use our `Import` function
	vm := makeVM()
	vm.Importer(r)

	// TODO This is largely implementing legacy Konjure behavior, is it still valid?
	var data string
	var err error
	if r.Filename != "" {
		data, err = vm.EvaluateFile(r.Filename)
	} else {
		filename := "<cmdline>"
		if r.Snippet == "" {
			filename = "<empty>"
		}
		data, err = vm.EvaluateAnonymousSnippet(filename, r.Snippet)
	}
	if err != nil {
		return nil, err
	}

	return r.parseJSON(data)
}

func (r *JsonnetReader) Import(importedFrom, importedPath string) (jsonnet.Contents, string, error) {
	return r.FileImporter.Import(importedFrom, importedPath)
}

// bundlerEnsure runs the Jsonnet bundler to ensure any dependencies are present.
func (r *JsonnetReader) bundlerEnsure() error {
	// Attempt to find and load the Jsonnet Bundler file
	jbfile, _, err := r.Import("", jsonnetfile.File)
	if err != nil {
		return err
	}
	jsonnetFile, err := jsonnetfile.Unmarshal([]byte(jbfile.String()))
	if err != nil {
		return err
	}

	// Attempt to find and load the Jsonnet Bundler lock file
	jblockfile, _, err := r.Import("", jsonnetfile.LockFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	lockFile, err := jsonnetfile.Unmarshal([]byte(jblockfile.String()))
	if err != nil {
		return err
	}

	// Create a temporary directory for the bundler to use
	if err := os.MkdirAll(filepath.Join(r.JsonnetBundlerPackageHome, ".tmp"), os.ModePerm); err != nil {
		return err
	}

	// TODO Append JsonnetBundlerPackageHome to the r.FileImporter JPath iff it is not already there

	// Ignore output when ensuring dependencies are updated
	color.Output = io.Discard

	_, err = jb.Ensure(jsonnetFile, r.JsonnetBundlerPackageHome, lockFile.Dependencies)
	return err
}

// parseJson takes Jsonnet output and makes it into resource nodes
func (r *JsonnetReader) parseJSON(j string) ([]*yaml.RNode, error) {
	t, err := json.NewDecoder(strings.NewReader(j)).Token()
	if err != nil {
		return nil, err
	}

	// Setup a byte reader
	br := &kio.ByteReader{SetAnnotations: map[string]string{}}

	// If it looks like a JSON list, trick the parser it by wrapping it with an items field
	if t == json.Delim('[') {
		br.Reader = strings.NewReader(`{"kind":"List","items":` + j + `}`)
		return br.Read()
	}

	if t != json.Delim('{') {
		return nil, fmt.Errorf("expected JSON object or list")
	}

	// Parse the JSON output as YAML
	br.Reader = strings.NewReader(j)
	result, err := br.Read()

	// Either an error, an empty document, or a `kind: List` that was unwrapped
	if err != nil || len(result) != 1 {
		return result, err
	}

	// The only object in the result is already a resource, just return
	if _, err := result[0].GetValidatedMetadata(); err == nil {
		return result, nil
	}

	// Convert the map of filename->documents into a list where each document has the name as a comment
	if err := result[0].VisitFields(func(node *yaml.MapNode) error {
		node.Value.YNode().HeadComment = fmt.Sprintf("Source: %s", node.Key.YNode().Value)
		result = append(result, node.Value)
		return nil
	}); err != nil {
		return nil, err
	}

	return result[1:], nil
}

// processParameters is a helper to configure the Jsonnet VM.
func processParameters(params []konjurev1beta2.JsonnetParameter, handleVar func(string, string), handleCode func(string, string)) {
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
