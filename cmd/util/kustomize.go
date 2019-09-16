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

package util

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

const (
	group = "konjure.carbonrelay.com"
)

// ExecPlugin implementations can be made into commands
// TODO Is this really GeneratorExecPlugin? Or is ExecPlugin one thing and Generator/Transformer another (e.g. Run accepts a resmap/reader)
type ExecPlugin interface {
	Unmarshal(y []byte, version string) error
	PreRun() error
	Run(cmd *cobra.Command, name string) error
}

// TODO Do we need a version of this that takes a Kustomize Transformer?
// TODO Or Should we have NewGeneratorCommand and NewTransformerCommand

func NewExecPluginCommand(kind string, p ExecPlugin) *cobra.Command {
	// TODO Any generic short/long/example text?
	return &cobra.Command{
		Use:  kind + " FILE",
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return p.PreRun()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := ioutil.ReadFile(args[0])
			if err != nil {
				return err
			}
			ver, name, err := checkConfig(cfg, kind)
			if err != nil {
				return err
			}
			err = p.Unmarshal(cfg, ver)
			if err != nil {
				return err
			}
			return p.Run(cmd, name)
		},
	}
}

// Checks the supplied plugin configuration, returning the extracted API version (not group) and metadata name
func checkConfig(b []byte, kind string) (string, string, error) {
	type Metadata struct {
		Name string `json:"name"`
	}
	type Config struct {
		APIVersion string   `json:"apiVersion"`
		Kind       string   `json:"kind"`
		Metadata   Metadata `json:"metadata"`
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return "", "", err
	}

	// Verify the API group and extract the version (so ExecPlugin implementations can convert if necessary)
	p := strings.Split(cfg.APIVersion, "/")
	if len(p) != 2 || p[0] != group {
		return "", "", fmt.Errorf("apiVersion should be %s", group)
	}

	// Verify the kind matches what was expected for this exec plugin
	if cfg.Kind != "" && cfg.Kind != kind {
		return "", "", fmt.Errorf("kind should be %s", kind)
	}

	return p[1], cfg.Metadata.Name, nil
}
