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

package jsonnet

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"unicode"

	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
)

// Parameter defines either and external variable or top-level argument; except name, all are mutually exclusive.
type Parameter struct {
	Name       string `json:"name,omitempty"`
	String     string `json:"string,omitempty"`
	StringFile string `json:"stringFile,omitempty"`
	Code       string `json:"code,omitempty"`
	CodeFile   string `json:"codeFile,omitempty"`
}

// Executor specifies configuration and execution helpers for running Jsonnet
type Executor struct {
	Bin    string `json:"bin,omitempty"`
	Stderr func() io.Writer
}

// Complete fills in the blank configuration values
func (jsonnet *Executor) Complete() {
	var err error

	if jsonnet.Bin == "" {
		if jsonnet.Bin, err = exec.LookPath("jsonnet"); err != nil {
			jsonnet.Bin = "jsonnet"
		}
	}
}

func (jsonnet *Executor) command(jpath []string, ext, tla []Parameter, extraArgs ...string) *exec.Cmd {
	var args []string

	for i := range ext {
		args = ext[i].AppendArgs(args, "--ext-")
	}

	for i := range tla {
		args = tla[i].AppendArgs(args, "--tla-")
	}

	for _, dir := range jpath {
		args = append(args, "--jpath", dir)
	}

	cmd := exec.Command(jsonnet.Bin, append(args, extraArgs...)...)
	return cmd
}

// ExecuteCode executes a snippet of Jsonnet code, returning a resource map
func (jsonnet *Executor) ExecuteCode(code string, path []string, ext, tla []Parameter) ([]byte, error) {
	b := &bytes.Buffer{}
	cmd := jsonnet.command(path, ext, tla, "--exec", "--", code)
	cmd.Stdout = b
	if jsonnet.Stderr != nil {
		cmd.Stderr = jsonnet.Stderr()
	}

	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// ExecuteFile executes a Jsonnet file, returning a resource map
func (jsonnet *Executor) ExecuteFile(filename string, path []string, ext, tla []Parameter) ([]byte, error) {
	b := &bytes.Buffer{}
	cmd := jsonnet.command(path, ext, tla, "--", filename)
	cmd.Stdout = b
	if jsonnet.Stderr != nil {
		cmd.Stderr = jsonnet.Stderr()
	}

	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// AppendArgs adds the Jsonnet command arguments corresponding to this value, prefix should be "--tla-" or "--ext-"
func (p *Parameter) AppendArgs(args []string, prefix string) []string {
	var opt, val string
	if p.String != "" {
		opt = "str"
		val = p.String
	} else if p.StringFile != "" {
		opt = "str-file"
		val = p.StringFile
	} else if p.Code != "" {
		opt = "code"
		val = p.Code
	} else if p.CodeFile != "" {
		opt = "code-file"
		val = p.CodeFile
	} else {
		return args
	}

	return append(args, prefix+opt, fmt.Sprintf("%s=%s", p.Name, val))
}

// AppendMultiDocumentJSONBytes inspects the supplied byte array to determine how it should be handled: if it
// is a JSON list, each item in the list is added to a new resource map; if the the command produces an object with a
// "kind" field, the contents are passed directly into the resource map; objects without a "kind" field are assumed
// to be a map of file names to document  contents and each field value is inserted to a new resource map honoring
// the order imposed by a sort of the keys.
func AppendMultiDocumentJSONBytes(rf *resource.Factory, m resmap.ResMap, b []byte) error {
	// This is JSON, we can trim the leading space
	j := bytes.TrimLeftFunc(b, unicode.IsSpace)
	if len(j) == 0 {
		return nil
	}

	if bytes.HasPrefix(j, []byte("[")) {
		// JSON list: just add each item as a new resource
		raw := make([]interface{}, 0)
		if err := json.Unmarshal(j, &raw); err != nil {
			return err
		}
		for i := range raw {
			if o, ok := raw[i].(map[string]interface{}); ok {
				if err := m.Append(rf.FromMap(o)); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("expected a list of objects")
			}
		}
		return nil
	}

	if bytes.HasPrefix(j, []byte("{")) {
		// JSON object: look for a "kind" field
		raw := make(map[string]interface{})
		if err := json.Unmarshal(j, &raw); err != nil {
			return err
		}
		if _, ok := raw["kind"]; ok {
			// If there is a "kind" field, assume the factory will know what to do with it
			if err := m.Append(rf.FromMap(raw)); err != nil {
				return err
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
						return err
					}
				} else {
					return fmt.Errorf("expected a map of objects")
				}
			}
		}
		return nil
	}

	return fmt.Errorf("expected JSON object or list")
}
