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

package helm

import (
	"bytes"
	"io"

	"github.com/spf13/cobra"
)

// Unlike real Helm, we don't preserve order of set flags!

func NewHelmCommand() *cobra.Command {
	hc := &helmCommand{
		options: NewHelmOptions(),
	}

	cmd := &cobra.Command{
		Use:    "helm CHART",
		Short:  "Inflate a Helm chart",
		Args:   cobra.ExactArgs(1),
		PreRun: hc.preRun,
		RunE:   hc.run,
	}

	cmd.Flags().BoolVar(&hc.options.IncludeTests, "include-tests", false, "do not remove resources labeled as test hooks")

	// These flags match what real Helm has
	cmd.Flags().StringVar(&hc.options.Repository, "repo", "", "repository `url` used to locate the chart")
	cmd.Flags().StringVarP(&hc.options.ReleaseName, "name", "n", "release-name", "release `name`")
	cmd.Flags().StringVar(&hc.options.Version, "version", "", "fetch a specific `version` of a chart; if empty, the latest version of the chart will be used")
	cmd.Flags().StringVar(&hc.options.Helm.Home, "home", "", "override the location of your Helm config `directory`")
	cmd.Flags().StringToStringVar(&hc.set, "set", nil, "set `value`s on the command line")
	cmd.Flags().StringToStringVar(&hc.setFile, "set-file", nil, "set values from `file`s on the command line")
	cmd.Flags().StringToStringVar(&hc.setString, "set-string", nil, "set string `value`s on the command line")
	cmd.Flags().StringArrayVarP(&hc.values, "values", "f", nil, "specify values in a YAML `file`")

	return cmd
}

type helmCommand struct {
	options   *HelmOptions
	set       map[string]string
	setFile   map[string]string
	setString map[string]string
	values    []string
}

// Convert the compatibility options to real options
func (hc *helmCommand) preRun(cmd *cobra.Command, args []string) {
	hc.options.Helm.Complete()

	if len(args) > 0 {
		hc.options.Chart = args[0]
	}

	for k, v := range hc.set {
		hc.options.Values = append(hc.options.Values, HelmValue{Name: k, Value: v})
	}

	for k, v := range hc.setFile {
		hc.options.Values = append(hc.options.Values, HelmValue{Name: k, Value: v, LoadFile: true})
	}

	for k, v := range hc.setString {
		hc.options.Values = append(hc.options.Values, HelmValue{Name: k, Value: v, ForceString: true})
	}

	for _, valueFile := range hc.values {
		hc.options.Values = append(hc.options.Values, HelmValue{File: valueFile})
	}
}

func (hc *helmCommand) run(cmd *cobra.Command, args []string) error {
	b, err := hc.options.Run()
	if err != nil {
		return err
	}

	_, err = io.Copy(cmd.OutOrStdout(), bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	return nil
}
