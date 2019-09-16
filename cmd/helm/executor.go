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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
)

// HelmValue specifies the source for chart configurations
type HelmValue struct {
	File        string      `json:"file,omitempty"`
	Name        string      `json:"name,omitempty"`
	Value       interface{} `json:"value,omitempty"`
	ForceString bool        `json:"forceString,omitempty"`
	LoadFile    bool        `json:"loadFile,omitempty"`
}

// Helm specifies configuration and execution helpers for running Helm in the context of fetching and rendering charts
type Helm struct {
	Bin          string `json:"bin,omitempty"`
	Home         string `json:"home,omitempty"`
	ArchiveCache string `json:"chartDir,omitempty"`
	// TODO Additional repositories?
}

// Complete fills in the blank configuration values
func (helm *Helm) Complete() {
	var err error

	// Lookup Helm on the PATH; default to "helm"
	if helm.Bin == "" {
		if helm.Bin, err = exec.LookPath("helm"); err != nil {
			helm.Bin = "helm"
		}
	}

	// Lookup the Helm home directory; default to "~/.helm" or "./.helm"
	if helm.Home == "" {
		cmd := exec.Command(helm.Bin, "home")
		if out, err := cmd.CombinedOutput(); err != nil {
			helm.Home = os.Getenv("HELM_HOME")
			if helm.Home == "" {
				helm.Home = filepath.Join(os.Getenv("HOME"), ".helm")
			}
		} else {
			helm.Home = strings.TrimSpace(string(out))
		}
	}

	// Default the "archive cache" directory inside Helm home
	if helm.ArchiveCache == "" {
		helm.ArchiveCache = filepath.Join(helm.Home, "cache", "archive")
	}
}

func (helm *Helm) command(args ...string) *exec.Cmd {
	cmd := exec.Command(helm.Bin, args...)
	cmd.Env = append(cmd.Env, "HELM_HOME="+helm.Home)
	return cmd
}

// Init runs a silent, client only, initialization
func (helm *Helm) Init() error {
	return helm.command("init", "--client-only").Run()
}

// Fetch downloads a chart with an optional specific version (leave version empty to get the latest version).
// The name of downloaded chart file is returned.
func (helm *Helm) Fetch(chart, version string) (string, error) {
	// Create a temporary directory for downloading since `helm fetch` won't tell us the name of the file
	d, err := ioutil.TempDir("", "helm-fetch-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(d) }()

	// Run the fetch command into the temporary directory
	var args []string
	args = append(args, "fetch", chart, "--destination", d)
	if version != "" {
		args = append(args, "--version", version)
	}
	if err := helm.command(args...).Run(); err != nil {
		return "", err
	}

	// Find the file and move it out of the temporary directory (overwriting existing files)
	if files, err := ioutil.ReadDir(d); err == nil && len(files) == 1 {
		filename := filepath.Join(helm.ArchiveCache, files[0].Name())
		if err := os.Remove(filename); err != nil && !os.IsNotExist(err) {
			return "", err
		}

		err := os.Rename(filepath.Join(d, files[0].Name()), filename)
		return filename, err
	}

	return "", fmt.Errorf("unable to find fetched chart")
}

// Template renders a chart archive using the specified release name and value overrides
func (helm *Helm) Template(filename string, name string, values []HelmValue) (resmap.ResMap, error) {
	// Construct the arguments
	var args []string
	args = append(args, "template", filename)
	if name != "" {
		args = append(args, "--name", name)
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

	rmf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), nil)
	return rmf.NewResMapFromBytes(b.Bytes())
}

// AppendArgs adds the Helm command arguments corresponding to this value
func (v *HelmValue) AppendArgs(args []string) []string {
	if v.File != "" {
		args = append(args, "--value", v.File)
	} else if v.Name != "" {
		setOpt := "--set"
		if v.LoadFile {
			setOpt = "--set-file"
		} else if v.ForceString {
			setOpt = "--set-string"
		}
		args = append(args, setOpt, fmt.Sprintf("%s=%s", v.Name, v.Value))
	}
	return args
}
