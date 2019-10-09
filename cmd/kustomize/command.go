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

package kustomize

import (
	"github.com/carbonrelay/konjure/cmd/berglas"
	"github.com/carbonrelay/konjure/cmd/helm"
	"github.com/carbonrelay/konjure/cmd/jsonnet"
	"github.com/carbonrelay/konjure/cmd/label"
	"github.com/spf13/cobra"
)

// The Kustomize command really just aggregates all the exec plugin commands in one place

// TODO Add documentation about how to use Konjure as a Kustomize plugin

func NewKustomizeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "kustomize",
	}

	cmd.AddCommand(newInitializeCommand())

	cmd.AddCommand(berglas.NewBerglasGenerator())
	cmd.AddCommand(berglas.NewBerglasTransformer())
	cmd.AddCommand(helm.NewHelmGenerator())
	cmd.AddCommand(jsonnet.NewJsonnetGenerator())
	cmd.AddCommand(label.NewLabelTransformer())

	return cmd
}

func newInitializeCommand() *cobra.Command {
	opts := NewInitializeOptions()

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Configure Kustomize",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.Complete()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVar(&opts.Source, "source", "", "Override the path to the source executable.")

	return cmd
}
