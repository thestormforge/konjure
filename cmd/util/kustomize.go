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

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	group = "konjure.carbonrelay.com"
)

// ConfigMetadata is the Kubernetes metadata associated with the configuration
type ConfigMetadata struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// ExecPlugin implementations can be made into commands
// TODO Is this really GeneratorExecPlugin? Or is ExecPlugin one thing and Generator/Transformer another (e.g. Run accepts a resmap/reader)
type ExecPlugin interface {
	Unmarshal(y []byte, metadata ConfigMetadata) error
	PreRun() error
	Run(cmd *cobra.Command) error
}

// TODO Do we need a version of this that takes a Kustomize Transformer?
// TODO Or Should we have NewGeneratorCommand and NewTransformerCommand

func NewExecPluginCommand(kind string, p ExecPlugin) *cobra.Command {
	// TODO Any generic short/long/example text?
	return &cobra.Command{
		Use:    kind + " FILE",
		Args:   cobra.ExactArgs(1),
		Hidden: true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return p.PreRun()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := ioutil.ReadFile(args[0])
			if err != nil {
				return err
			}

			md, err := checkConfig(cfg, kind)
			if err != nil {
				return err
			}

			err = p.Unmarshal(cfg, md)
			if err != nil {
				return err
			}

			return p.Run(cmd)
		},
	}
}

// Checks the supplied plugin configuration, returning the extracted API version (not group) and metadata name
func checkConfig(b []byte, kind string) (ConfigMetadata, error) {
	cfg := ConfigMetadata{}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}

	// Verify the API group independent of the version (so ExecPlugin implementations can convert if necessary)
	if cfg.GroupVersionKind().Group != group {
		return cfg, fmt.Errorf("group should be %s", group)
	}

	// Verify the kind matches what was expected for this exec plugin
	if cfg.Kind != "" && cfg.Kind != kind {
		return cfg, fmt.Errorf("kind should be %s", kind)
	}

	return cfg, nil
}
