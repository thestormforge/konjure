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

	"github.com/spf13/cobra"
)

func NewBerglasCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "berglas",
	}
	cmd.AddCommand(newGenerateCommand())
	cmd.AddCommand(newTransformCommand())
	return cmd
}

func newGenerateCommand() *cobra.Command {
	bc := &berglasGenerateCommand{
		options: NewBerglasGenerateOptions(),
	}

	cmd := &cobra.Command{
		Use:  "generate",
		RunE: bc.run,
	}

	cmd.Flags().StringVar(&bc.name, "name", "", "Name of the secret to generate.")
	cmd.Flags().StringArrayVarP(&bc.options.References, "ref", "R", nil, "Berglas references to include in the secret")

	return cmd
}

type berglasGenerateCommand struct {
	options *BerglasGenerateOptions
	name    string
}

func (bc *berglasGenerateCommand) run(cmd *cobra.Command, args []string) error {
	b, err := bc.options.Run(bc.name)
	if err != nil {
		return err
	}

	_, err = io.Copy(cmd.OutOrStdout(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	return nil
}

func newTransformCommand() *cobra.Command {
	bc := &berglasTransformCommand{
		options: NewBerglasTransformOptions(),
	}

	cmd := &cobra.Command{
		Use:  "transform",
		RunE: bc.run,
	}

	cmd.Flags().StringVarP(&bc.filename, "filename", "f", "", "File that contains the configuration to transform.")
	cmd.Flags().BoolVar(&bc.options.GenerateSecrets, "secrets", false, "Perform transformation using secrets.")

	return cmd
}

type berglasTransformCommand struct {
	options  *BerglasTransformOptions
	filename string
}

func (bc *berglasTransformCommand) run(cmd *cobra.Command, args []string) error {
	var in []byte
	var err error
	if bc.filename != "-" {
		in, err = ioutil.ReadFile(bc.filename)
	} else {
		in, err = ioutil.ReadAll(cmd.InOrStdin())
	}
	if err != nil {
		return err
	}

	b, err := bc.options.Run(in)
	if err != nil {
		return err
	}

	_, err = io.Copy(cmd.OutOrStdout(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	return nil
}
