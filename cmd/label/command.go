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

	"github.com/spf13/cobra"
)

func NewLabelCommand() *cobra.Command {
	lc := &labelCommand{
		options: NewLabelOptions(),
	}

	cmd := &cobra.Command{
		Use:  "label",
		RunE: lc.run,
	}

	cmd.Flags().StringVarP(&lc.filename, "filename", "f", "", "File that contains the configuration to transform.")
	cmd.Flags().StringToStringVarP(&lc.options.Labels, "label", "l", nil, "Common labels to add.")

	return cmd
}

type labelCommand struct {
	options  *LabelOptions
	filename string
}

func (lc *labelCommand) run(cmd *cobra.Command, args []string) error {
	var in []byte
	var err error
	if lc.filename != "-" {
		in, err = ioutil.ReadFile(lc.filename)
	} else {
		in, err = ioutil.ReadAll(cmd.InOrStdin())
	}
	if err != nil {
		return err
	}

	b, err := lc.options.Run(in)
	if err != nil {
		return err
	}

	_, err = io.Copy(cmd.OutOrStdout(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	return nil
}
