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

package helm

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
)

// Value specifies the source for chart configurations
type Value struct {
	File        string      `json:"file,omitempty"`
	Name        string      `json:"name,omitempty"`
	Value       interface{} `json:"value,omitempty"`
	ForceString bool        `json:"forceString,omitempty"`
	LoadFile    bool        `json:"loadFile,omitempty"`
}

// Executor specifies configuration and execution helpers for running Helm in the context of fetching and rendering charts
type Executor struct {
	Bin             string `json:"bin,omitempty"`
	RepositoryCache string `json:"repositoryCache,omitempty"`
}

func (helm *Executor) command(args ...string) *exec.Cmd {
	bin := helm.Bin
	if bin == "" {
		bin = "helm"
	}
	cmd := exec.Command(bin, args...)
	if helm.RepositoryCache != "" {
		cmd.Env = append(cmd.Env, "HELM_REPOSITORY_CACHE="+helm.RepositoryCache)
	}
	return cmd
}

// Template renders a chart archive using the specified release name and value overrides
func (helm *Executor) Template(name, chart, version, namespace, repo string, values []Value) ([]byte, error) {
	// Construct the arguments
	var args []string
	args = append(args, "template")
	if name != "" {
		args = append(args, name)
	} else {
		// TODO Does this always just produce "RELEASE-NAME"?
		args = append(args, "--generate-name")
	}
	args = append(args, chart)

	if version != "" {
		args = append(args, "--version", version)
	}
	if namespace != "" {
		args = append(args, "--namespace", namespace)
	}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	for i := range values {
		args = values[i].AppendArgs(args)
	}

	b := &bytes.Buffer{}
	cmd := helm.command(args...)
	cmd.Stdout = b

	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// AppendArgs adds the Helm command arguments corresponding to this value
func (v *Value) AppendArgs(args []string) []string {
	if v.File != "" {
		// Try to expand a glob; if it fails or does not match, pass on the raw value and let Helm figure it out
		valueFiles := []string{v.File}
		if matches, err := filepath.Glob(v.File); err == nil && len(matches) > 0 {
			valueFiles = matches
		}
		for _, f := range valueFiles {
			args = append(args, "--values", f)
		}
	} else if v.Name != "" {
		setOpt := "--set"
		if v.LoadFile {
			setOpt = "--set-file"
		} else if v.ForceString {
			setOpt = "--set-string"
		}
		args = append(args, setOpt, fmt.Sprintf("%s=%v", v.Name, v.Value))
	}
	return args
}
