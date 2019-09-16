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

package berglas

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/carbonrelay/konjure/cmd/util"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func NewBerglasGenerator() *cobra.Command {
	bg := &berglasGenerator{
		options: NewBerglasGenerateOptions(),
	}

	return util.NewExecPluginCommand("BerglasGenerator", bg)
}

type berglasGenerator struct {
	options *BerglasGenerateOptions
}

func (bg *berglasGenerator) Unmarshal(y []byte, version string) error {
	return yaml.Unmarshal(y, bg.options)
}

func (bg *berglasGenerator) PreRun() error {
	return nil
}

func (bg *berglasGenerator) Run(cmd *cobra.Command, name string) error {
	b, err := bg.options.Run(name)
	if err != nil {
		return err
	}

	_, err = io.Copy(cmd.OutOrStdout(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	return nil
}

func NewBerglasTransformer() *cobra.Command {
	bt := &berglasTransformer{
		options: NewBerglasTransformOptions(),
	}

	return util.NewExecPluginCommand("BerglasTransformer", bt)
}

type berglasTransformer struct {
	options *BerglasTransformOptions
}

func (bt *berglasTransformer) Unmarshal(y []byte, version string) error {
	return yaml.Unmarshal(y, bt.options)
}

func (bt *berglasTransformer) PreRun() error {
	return nil
}

func (bt *berglasTransformer) Run(cmd *cobra.Command, name string) error {
	in, err := ioutil.ReadAll(cmd.InOrStdin())
	if err != nil {
		return err
	}

	b, err := bt.options.Run(in)
	if err != nil {
		return err
	}

	_, err = io.Copy(cmd.OutOrStdout(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	return nil
}
