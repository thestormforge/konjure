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

	"github.com/carbonrelay/konjure/internal/env"
	"github.com/carbonrelay/konjure/internal/kustomize"
	"github.com/carbonrelay/konjure/internal/kustomize/edit"
	"github.com/carbonrelay/konjure/plugin/berglas"
	"github.com/carbonrelay/konjure/plugin/cat"
	"github.com/carbonrelay/konjure/plugin/filter"
	"github.com/carbonrelay/konjure/plugin/helm"
	"github.com/carbonrelay/konjure/plugin/jsonnet"
	"github.com/carbonrelay/konjure/plugin/label"
	"github.com/carbonrelay/konjure/plugin/random"
	"github.com/carbonrelay/konjure/plugin/secret"
	"github.com/spf13/cobra"
)

var version = "unspecified"

const (
	rootExample = `
# Use Konjure to render a Helm chart (requires 'helm' on your 'PATH')
konjure helm --name "my-release" ${CHART}

# Concatenate manifests into a single document stream
konjure cat manifest1.yaml manifest2.yaml

# Generate a Kubernetes secret using sensitive data stored using Berglas
konjure berglas generate --name "my-secret" --ref "berglas://${BUCKET_ID}/some-secret-key"

# Install Konjure as a series of Kustomize plugins
konjure kustomize init
`

	kustomizeExample = `
# Edit a kustomization to include a generator configuration file
# NOTE: This functionality will be removed when it makes it into Kustomize proper
konjure kustomize edit add generator my-konjure-plugin-config.yaml

# Install Konjure as a series of Kustomize plugins
konjure kustomize init
`
)

func main() {
	rootCmd := newRootCommand(os.Args[0])
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCommand(arg0 string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "konjure",
		Short:   "Manifest, appear!",
		Example: rootExample,
		Version: version,
	}

	kustomizeCmd := &cobra.Command{
		Use:     "kustomize",
		Short:   "Extensions for Kustomize",
		Example: kustomizeExample,
	}

	cobra.EnableCommandSorting = false
	addPlugins(rootCmd, kustomizeCmd)

	// If arg0 matches on the kustomizeCmd (i.e. via a symlink to the binary), return it instead
	if c, _, err := kustomizeCmd.Find([]string{filepath.Base(arg0)}); err == nil {
		kustomizeCmd.RemoveCommand(c)
		return c
	}

	// Add the remaining commands
	kustomizeCmd.AddCommand(kustomize.NewInitializeCommand())
	kustomizeCmd.AddCommand(edit.NewEditCommand())
	rootCmd.AddCommand(kustomizeCmd)

	return rootCmd
}

func addPlugins(rootCmd, kustomizeCmd *cobra.Command) {
	rootCmd.AddCommand(env.NewCommand())

	rootCmd.AddCommand(berglas.NewBerglasCommand())
	rootCmd.AddCommand(cat.NewCatCommand())
	rootCmd.AddCommand(filter.NewFilterCommand())
	rootCmd.AddCommand(helm.NewHelmCommand())
	rootCmd.AddCommand(jsonnet.NewJsonnetCommand())
	rootCmd.AddCommand(label.NewLabelCommand())
	rootCmd.AddCommand(random.NewRandomCommand())
	rootCmd.AddCommand(secret.NewSecretCommand())

	kustomizeCmd.AddCommand(berglas.NewBerglasGeneratorExecPlugin())
	kustomizeCmd.AddCommand(berglas.NewBerglasTransformerExecPlugin())
	kustomizeCmd.AddCommand(cat.NewCatGeneratorExecPlugin())
	kustomizeCmd.AddCommand(helm.NewHelmGeneratorExecPlugin())
	kustomizeCmd.AddCommand(jsonnet.NewJsonnetGeneratorExecPlugin())
	kustomizeCmd.AddCommand(label.NewLabelTransformerExecPlugin())
	kustomizeCmd.AddCommand(random.NewRandomGeneratorExecPlugin())
	kustomizeCmd.AddCommand(secret.NewSecretGeneratorExecPlugin())
}
