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

package konjure

import (
	"io"
	"os/exec"

	"github.com/thestormforge/konjure/internal/readers"
	"github.com/thestormforge/konjure/pkg/filters"
	"sigs.k8s.io/kustomize/kyaml/kio"
	kiofilters "sigs.k8s.io/kustomize/kyaml/kio/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Filter replaces Konjure resources with the expanded resources they represent.
type Filter struct {
	// The number of times to recursively filter the resource list.
	Depth int
	// The default reader to use, defaults to stdin.
	DefaultReader io.Reader
	// Filter used to reduce the output to application definitions.
	ApplicationFilter filters.ApplicationFilter
	// Filter used to reduce the output to workloads.
	WorkloadFilter filters.WorkloadFilter
	// Filter to determine which resources are retained.
	filters.ResourceMetaFilter
	// Flag indicating that status fields should not be stripped.
	KeepStatus bool
	// Flag indicating that comments should not be stripped.
	KeepComments bool
	// Flag indicating that style should be reset.
	ResetStyle bool
	// Flag indicating that output should be formatted.
	Format bool
	// Flag indicating that output should be sorted.
	Sort bool
	// Flag indicating that output should be reverse sorted (implies sort=true).
	Reverse bool
	// The explicit working directory used to resolve relative paths.
	WorkingDirectory string
	// Flag indicating we can process directories recursively.
	RecursiveDirectories bool
	// Kinds which should not be expanded (e.g. "Kustomize").
	DoNotExpand []string
	// Override the default path to the kubeconfig file.
	Kubeconfig string
	// Override the default types used when fetching Kubernetes resources.
	KubernetesTypes []string
	// Override the default Kubectl executor.
	KubectlExecutor func(cmd *exec.Cmd) ([]byte, error)
	// Override the default Kustomize executor.
	KustomizeExecutor func(cmd *exec.Cmd) ([]byte, error)
}

// Filter evaluates Konjure resources according to the filter configuration.
func (f *Filter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	defaultTypes := f.KubernetesTypes
	if len(defaultTypes) == 0 {
		// This represents the original set of default types from early StormForge products
		defaultTypes = append(defaultTypes, "deployments", "statefulsets", "configmaps")
	}

	if f.WorkloadFilter.Enabled {
		// Include the built-in workload types (and intermediaries necessary for detection to work)
		defaultTypes = appendDistinct(defaultTypes, "daemonsets", "deployments", "statefulsets", "replicasets", "cronjobs", "pods")
	}

	p := &filters.Pipeline{
		Inputs: []kio.Reader{kio.ResourceNodeSlice(nodes)},
		Filters: []kio.Filter{
			&readers.Filter{
				Depth: f.Depth,
				ReaderOptions: []readers.Option{
					readers.WithDefaultInputStream(f.DefaultReader),
					readers.WithWorkingDirectory(f.WorkingDirectory),
					readers.WithRecursiveDirectories(f.RecursiveDirectories),
					readers.WithKubeconfig(f.Kubeconfig),
					readers.WithKubectlExecutor(f.KubectlExecutor),
					readers.WithKustomizeExecutor(f.KustomizeExecutor),
					readers.WithDefaultTypes(defaultTypes...),
					readers.WithoutKindExpansion(f.DoNotExpand...),
				},
			},

			&f.ApplicationFilter,
			&f.WorkloadFilter,
			&f.ResourceMetaFilter,
		},
	}

	if !f.KeepStatus {
		p.Filters = append(p.Filters, kio.FilterAll(yaml.Clear("status")))
	}

	if !f.KeepComments {
		p.Filters = append(p.Filters, &kiofilters.StripCommentsFilter{})
	}

	if f.ResetStyle {
		p.Filters = append(p.Filters, kio.FilterAll(filters.ResetStyle()))
	}

	if f.Format {
		p.Filters = append(p.Filters, &kiofilters.FormatFilter{})
	}

	if f.Reverse {
		p.Filters = append(p.Filters, filters.UninstallOrder())
	} else if f.Sort {
		p.Filters = append(p.Filters, filters.InstallOrder())
	}

	return p.Read()
}

func appendDistinct(values []string, more ...string) []string {
	contains := make(map[string]struct{}, len(values)+len(more))
	for _, v := range values {
		contains[v] = struct{}{}
	}
	for _, m := range more {
		if _, ok := contains[m]; !ok {
			contains[m] = struct{}{}
			values = append(values, m)
		}
	}
	return values
}
