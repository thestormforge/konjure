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

package label

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/carbonrelay/konjure/cmd/util"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func NewLabelTransformer() *cobra.Command {
	lt := &labelTransformer{
		options: NewLabelOptions(),
	}

	return util.NewExecPluginCommand("konjure.carbonrelay.com", "v1beta1", "LabelTransformer", lt)
}

type labelTransformer struct {
	options *LabelOptions
}

func (lt *labelTransformer) Unmarshal(y []byte, metadata util.ConfigMetadata) error {
	return yaml.Unmarshal(y, lt.options)
}

func (lt *labelTransformer) PreRun() error {
	return nil
}

func (lt *labelTransformer) Run(cmd *cobra.Command) error {
	in, err := ioutil.ReadAll(cmd.InOrStdin())
	if err != nil {
		return err
	}

	b, err := lt.options.Run(in)
	if err != nil {
		return err
	}

	_, err = io.Copy(cmd.OutOrStdout(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	return nil
}
