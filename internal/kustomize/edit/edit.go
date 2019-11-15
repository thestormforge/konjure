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

package edit

import (
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/api/filesys"
)

// NewEditCommand returns a minimal edit command with the Konjure specific additions
func NewEditCommand() *cobra.Command {
	fSys := filesys.MakeFsOnDisk()
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Adds an item to the kustomization file.",
		Args:  cobra.MinimumNArgs(1),
	}
	addCmd.AddCommand(newCmdAddGenerator(fSys), newCmdAddTransformer(fSys))

	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "Removes items from the kustomization file.",
		Args:  cobra.MinimumNArgs(1),
	}
	removeCmd.AddCommand(newCmdRemoveGenerator(fSys), newCmdRemoveTransformer(fSys))

	editCmd := &cobra.Command{
		Use:   "edit",
		Short: "Edits a kustomization file",
		Long:  "Edits a kustomization file; this functionality is only provided as a stop-gap until it available in Kustomize",
		Args:  cobra.MinimumNArgs(1),
	}
	editCmd.AddCommand(addCmd, removeCmd)
	return editCmd
}
