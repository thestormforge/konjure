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
	"io"

	"github.com/carbonrelay/konjure/cmd/util"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func NewHelmGenerator() *cobra.Command {
	hep := &helmGenerator{
		options: NewHelmOptions(),
	}

	return util.NewExecPluginCommand("konjure.carbonrelay.com", "v1beta1", "HelmGenerator", hep)
}

type helmGenerator struct {
	options *HelmOptions
}

func (hg *helmGenerator) Unmarshal(y []byte, metadata util.ConfigMetadata) error {
	hg.options.ReleaseName = metadata.Name
	return yaml.Unmarshal(y, hg.options)
}

func (hg *helmGenerator) PreRun() error {
	hg.options.Helm.Complete()
	return nil
}

func (hg *helmGenerator) Run(cmd *cobra.Command) error {
	b, err := hg.options.Run()
	if err != nil {
		return err
	}

	_, err = io.Copy(cmd.OutOrStdout(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	return nil
}
