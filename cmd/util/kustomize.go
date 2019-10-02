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

// ConfigMetadata is the Kubernetes metadata associated with the configuration
type ConfigMetadata struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// TODO Should we leverage the Kustomize APIs instead of our own ExecPlugin interface?

// ExecPlugin implementations can be made into commands
type ExecPlugin interface {
	Unmarshal(y []byte, metadata ConfigMetadata) error
	PreRun() error
	Run(cmd *cobra.Command) error
}

// NewExecPluginCommand returns a command for the supplied executable plugin
func NewExecPluginCommand(group, version, kind string, p ExecPlugin) *cobra.Command {
	// TODO Any generic short/long/example text?
	return &cobra.Command{
		Use:         kind + " FILE",
		Version:     version,
		Annotations: map[string]string{"group": group},
		Args:        cobra.ExactArgs(1),
		Hidden:      true,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return p.PreRun()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := ioutil.ReadFile(args[0])
			if err != nil {
				return err
			}

			md, err := checkConfig(cmd, cfg)
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

// Returns the metadata extracted from the supplied configuration after verifying it against the supplied command
func checkConfig(cmd *cobra.Command, b []byte) (ConfigMetadata, error) {
	cfg := ConfigMetadata{}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}

	// Get the GVK of the object we just unmarshalled, it should match the command
	gvk := cfg.GroupVersionKind()

	// Verify the API group independent of the version (so ExecPlugin implementations can convert if necessary)
	if gvk.Group != "" && gvk.Group != cmd.Annotations["group"] {
		return cfg, fmt.Errorf("group should be %s", cmd.Annotations["group"])
	}

	// TODO Verify the version? Support some type of conversion?

	// Verify the kind matches what was expected for this exec plugin
	if gvk.Kind != "" && gvk.Kind != cmd.Name() {
		return cfg, fmt.Errorf("kind should be %s", cmd.Name())
	}

	return cfg, nil
}
