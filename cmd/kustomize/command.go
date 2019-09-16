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
	"github.com/spf13/cobra"
)

// The Kustomize command really just aggregates all the exec plugin commands in one place

// TODO Add documentation about how to use Konjure as a Kustomize plugin

func NewKustomizeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "kustomize",
		Hidden: true,
	}

	cmd.AddCommand(berglas.NewBerglasGenerator())
	cmd.AddCommand(berglas.NewBerglasTransformer())
	cmd.AddCommand(helm.NewHelmGenerator())
	cmd.AddCommand(jsonnet.NewJsonnetGenerator())

	return cmd
}
