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
	"io"

	"github.com/carbonrelay/konjure/cmd/util"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func NewJsonnetGenerator() *cobra.Command {
	jg := &jsonnetGenerator{
		options: NewJsonnetOptions(),
	}

	return util.NewExecPluginCommand("JsonnetGenerator", jg)
}

type jsonnetGenerator struct {
	options *JsonnetOptions
}

func (jg *jsonnetGenerator) Unmarshal(y []byte, version string) error {
	return yaml.Unmarshal(y, jg.options)
}

func (jg *jsonnetGenerator) PreRun() error {
	jg.options.Jsonnet.Complete()
	return nil
}

func (jg *jsonnetGenerator) Run(cmd *cobra.Command, name string) error {
	b, err := jg.options.Run(cmd.ErrOrStderr())
	if err != nil {
		return err
	}

	_, err = io.Copy(cmd.OutOrStdout(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	return nil
}
