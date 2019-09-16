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
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
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

// Jsonnet specifies configuration and execution helpers for running Jsonnet
type Jsonnet struct {
	Bin         string `json:"bin,omitempty"`
	DockerImage string `json:"image,omitempty"`
}

// Complete fills in the blank configuration values
func (jsonnet *Jsonnet) Complete() {
	var err error

	if jsonnet.Bin == "" {
		if jsonnet.Bin, err = exec.LookPath("jsonnet"); err != nil {
			// If Jsonnet is not available, we may be able to run it through Docker instead
			if dockerName, dockerErr := exec.LookPath("docker"); dockerErr != nil {
				// No Docker either
				jsonnet.Bin = "jsonnet"
			} else {
				var cmd *exec.Cmd
				if jsonnet.DockerImage != "" {
					// Pull now to avoid unsuppressable noise later
					cmd = exec.Command(dockerName, "image", "pull", jsonnet.DockerImage)
				} else {
					// The official Jsonnet repository Docker image does not seem to be published, instead of pulling, build the image
					repo, tag := "google/jsonnet", "0.14.0"
					jsonnet.DockerImage = fmt.Sprintf("x-%s:%s", repo, tag)
					// TODO Skip the build if the image exists
					cmd = exec.Command(dockerName, "image", "build", "--tag", jsonnet.DockerImage, fmt.Sprintf("https://github.com/%s.git#v%s", repo, tag))
				}

				if err := cmd.Run(); err != nil {
					// Failed to get a local copy of the image
					jsonnet.Bin = "jsonnet"
					jsonnet.DockerImage = ""
				} else {
					// We can successfully use the Docker executable
					jsonnet.Bin = dockerName
				}
			}
		} else {
			// Clear the Docker image name so we do not try to use it
			jsonnet.DockerImage = ""
		}
	}
}

func (jsonnet *Jsonnet) command(jpath []string, ext, tla []Parameter, extraArgs ...string) *exec.Cmd {
	var args []string

	if jsonnet.DockerImage != "" {
		// Modify the arguments for a Docker run
		args = append(args, "container", "run", "--rm")
		// TODO Mount jpath/modify the extraArg file name
		args = append(args, jsonnet.DockerImage)
	}

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
func (jsonnet *Jsonnet) ExecuteCode(code string, path []string, ext, tla []Parameter, stderr io.Writer) (resmap.ResMap, error) {
	stdout := &bytes.Buffer{}
	cmd := jsonnet.command(path, ext, tla, "--exec", "--", code)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return newResMapFromMultiDocumentJSONBytes(stdout.Bytes())
}

// ExecuteFile executes a Jsonnet file, returning a resource map
func (jsonnet *Jsonnet) ExecuteFile(filename string, path []string, ext, tla []Parameter, stderr io.Writer) (resmap.ResMap, error) {
	stdout := &bytes.Buffer{}
	cmd := jsonnet.command(path, ext, tla, "--", filename)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return newResMapFromMultiDocumentJSONBytes(stdout.Bytes())
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

// newResMapFromMultiDocumentJSONBytes inspects the supplied byte array to determine how it should be handled: if it
// is a JSON list, each item in the list is added to a new resource map; if the the command produces an object with a
// "kind" field, the contents are passed directly into the resource map; objects without a "kind" field are assumed
// to be a map of file names to document  contents and each field value is inserted to a new resource map honoring
// the order imposed by a sort of the keys.
func newResMapFromMultiDocumentJSONBytes(b []byte) (resmap.ResMap, error) {
	// Allocate a new resource map that we can append into
	rf := resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl())
	m := resmap.New()

	// This is JSON, we can trim the leading space
	j := bytes.TrimLeftFunc(b, unicode.IsSpace)

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
	} else if bytes.HasPrefix(j, []byte("{")) {
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
	} else {
		return nil, fmt.Errorf("expected JSON object or list")
	}

	return m, nil
}
