/*
Copyright 2021 GramLabs, Inc.

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
	"os"

	"github.com/spf13/cobra"
	"github.com/thestormforge/konjure/pkg/konjure"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
)

func NewRootCommand(version, refspec, date string) *cobra.Command {
	r := konjure.Resources{}
	f := &konjure.Filter{}
	w := &konjure.Writer{}

	cmd := &cobra.Command{
		Use:              "konjure INPUT...",
		Short:            "Manifest, appear!",
		Version:          version,
		SilenceUsage:     true,
		TraverseChildren: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"yaml", "yml", "json"}, cobra.ShellCompDirectiveFilterFileExt
		},
		Annotations: map[string]string{
			"BuildRefspec": refspec,
			"BuildDate":    date,
		},
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			w.Writer = cmd.OutOrStdout()
			f.DefaultReader = cmd.InOrStdin()

			if len(args) > 0 {
				r = append(r, konjure.NewResource(args...))
			} else {
				r = append(r, konjure.NewResource("-"))
			}

			f.WorkingDirectory, err = os.Getwd()

			if !w.KeepReaderAnnotations {
				w.ClearAnnotations = append(w.ClearAnnotations,
					kioutil.PathAnnotation,
					kioutil.LegacyPathAnnotation,
					filters.FmtAnnotation,
				)
			}

			return
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return kio.Pipeline{
				Inputs:                []kio.Reader{r},
				Filters:               []kio.Filter{f},
				Outputs:               []kio.Writer{w},
				ContinueOnEmptyResult: true,
			}.Execute()
		},
	}

	cmd.Flags().IntVarP(&f.Depth, "depth", "d", 100, "limit the number of times expansion can happen")
	cmd.Flags().StringVarP(&f.LabelSelector, "selector", "l", "", "label query to filter on")
	cmd.Flags().StringVar(&f.Kind, "kind", "", "keep only resource matching the specified kind")
	cmd.Flags().BoolVar(&f.KeepStatus, "keep-status", false, "retain status fields, if present")
	cmd.Flags().BoolVar(&f.KeepComments, "keep-comments", true, "retain YAML comments")
	cmd.Flags().BoolVar(&f.Format, "format", false, "format output to Kubernetes conventions")
	cmd.Flags().BoolVar(&w.RestoreVerticalWhiteSpace, "vws", false, "attempt to restore vertical white space")
	cmd.Flags().BoolVarP(&f.RecursiveDirectories, "recurse", "r", false, "recursively process directories")
	cmd.Flags().StringVar(&f.Kubeconfig, "kubeconfig", "", "path to the kubeconfig file")
	cmd.Flags().StringVarP(&w.Format, "output", "o", "yaml", "set the output format (yaml, json, ndjson, env, name, columns=, csv=, template=)")
	cmd.Flags().BoolVar(&w.KeepReaderAnnotations, "keep-annotations", false, "retain annotations used for processing")
	cmd.Flags().BoolVar(&f.Sort, "sort", false, "sort output prior to writing")
	cmd.Flags().BoolVar(&f.Reverse, "reverse", false, "reverse sort output prior to writing")
	cmd.Flags().BoolVar(&f.ApplicationFilter.Enabled, "apps", false, "transform output to application definitions")
	cmd.Flags().StringSliceVar(&f.ApplicationFilter.ApplicationNameLabels, "application-name-label", nil, "label to use for application names")
	cmd.Flags().BoolVar(&f.WorkloadFilter.Enabled, "workloads", false, "keep only workload resources")

	_ = cmd.Flags().MarkHidden("apps")                   // TODO This is "early access"
	_ = cmd.Flags().MarkHidden("application-name-label") // TODO This is "early access"
	_ = cmd.Flags().MarkHidden("workloads")              // TODO This is "early access"
	_ = cmd.Flags().MarkHidden("vws")                    // TODO This is "early access" / "somewhat unstable"

	cmd.AddCommand(
		NewHelmCommand(),
		NewHelmValuesCommand(),
		NewJsonnetCommand(),
		NewSecretCommand(),
	)

	return cmd
}
