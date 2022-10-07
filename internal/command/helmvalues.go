/*
Copyright 2022 GramLabs, Inc.

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

package command

import (
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/thestormforge/konjure/pkg/filters"
	"github.com/thestormforge/konjure/pkg/konjure"
	"github.com/thestormforge/konjure/pkg/pipes"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func NewHelmValuesCommand() *cobra.Command {
	var (
		valueOptions pipes.HelmValues
		schema       string
		w            konjure.GroupWriter
		inPlace      bool
	)

	cmd := &cobra.Command{
		Use:     "helm-values FILE ...",
		Short:   "Merge Helm values.yaml files",
		Aliases: []string{"values"},
	}

	cmd.Flags().StringSliceVarP(&valueOptions.ValueFiles, "values", "f", []string{}, "specify values in a YAML `file` (can specify multiple)")
	cmd.Flags().StringArrayVar(&valueOptions.Values, "set", []string{}, "set values on the command line (for example, `key1=val1`,key2=val2,...)")
	cmd.Flags().StringArrayVar(&valueOptions.StringValues, "set-string", []string{}, "set STRING values on the command line (for example, `key1=val1`,key2=val2,...)")
	cmd.Flags().StringArrayVar(&valueOptions.FileValues, "set-file", []string{}, "set values from respective files specified via the command line (for example, `key1=path1`,key2=path2,...)")
	cmd.Flags().StringVar(&schema, "schema", "", "the values.schema.json `file`; only necessary if it includes Kubernetes extensions with merge instructions")
	cmd.Flags().BoolVarP(&inPlace, "in-place", "i", false, "edit files in-place (if multiple FILE arguments are supplied, only the last file is overwritten)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Configure the writer to overwrite
		w.GroupWriter = func(name string) (io.Writer, error) {
			if inPlace {
				return os.Create(name)
			}
			return &nopCloser{Writer: cmd.OutOrStdout()}, nil
		}

		// Load a subset of the values.schema.json file for merging (if provided)
		var s *spec.Schema
		if schema != "" {
			data, err := os.ReadFile(schema)
			if err != nil {
				return err
			}
			s = &spec.Schema{}
			if err := s.UnmarshalJSON(data); err != nil {
				return err
			}
		}

		// The file _arguments_ are merged first (i.e. including comments), while `--values` files are just merged together
		return kio.Pipeline{
			Inputs:  append(pipes.CommandReaders(cmd, args), &valueOptions),
			Filters: []kio.Filter{filters.Flatten(s)},
			Outputs: []kio.Writer{&w},
		}.Execute()
	}
	return cmd
}

// nopCloser is used to block callers from closing stdout.
type nopCloser struct{ io.Writer }

// Close does nothing.
func (nopCloser) Close() error { return nil }
