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

package initialize

import "github.com/spf13/cobra"

func NewInitializeCommand() *cobra.Command {
	ic := &initializeCommand{
		options: NewInitializeOptions(),
	}

	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Configure Kustomize",
		PreRunE: ic.preRun,
		RunE:    ic.run,
	}

	cmd.Flags().StringVar(&ic.options.Source, "source", "", "Override the path to the source executable.")
	cmd.Flags().BoolVar(&ic.options.DryRun, "dry-run", false, "Display links instead of creating them.")

	return cmd
}

type initializeCommand struct {
	options *InitializeOptions
}

func (ic *initializeCommand) preRun(cmd *cobra.Command, args []string) error {
	return ic.options.Complete()
}

func (ic *initializeCommand) run(cmd *cobra.Command, args []string) error {
	return ic.options.Run(cmd.OutOrStdout())
}
