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

package main

import (
	"os"
	"path/filepath"

	"github.com/carbonrelay/konjure/cmd/berglas"
	"github.com/carbonrelay/konjure/cmd/helm"
	"github.com/carbonrelay/konjure/cmd/initialize"
	"github.com/carbonrelay/konjure/cmd/jsonnet"
	"github.com/carbonrelay/konjure/cmd/kustomize"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := NewRootCommand(os.Args[0])
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func NewRootCommand(arg0 string) *cobra.Command {
	// Check to see if we should use one of the Kustomize sub-commands directly
	kustomizeCommand := kustomize.NewKustomizeCommand()
	if c, _, err := kustomizeCommand.Find([]string{filepath.Base(arg0)}); err == nil {
		kustomizeCommand.RemoveCommand(c)
		return c
	}

	// Build the real root command
	rootCmd := &cobra.Command{
		Use:   "konjure",
		Short: "Manifest, appear!",
	}

	rootCmd.AddCommand(kustomizeCommand)
	rootCmd.AddCommand(initialize.NewInitializeCommand())

	rootCmd.AddCommand(berglas.NewBerglasCommand())
	rootCmd.AddCommand(helm.NewHelmCommand())
	rootCmd.AddCommand(jsonnet.NewJsonnetCommand())

	return rootCmd
}
