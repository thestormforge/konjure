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
	// Filter to determine which resources are retained.
	filters.ResourceMetaFilter
	// Flag indicating that status fields should not be stripped.
	KeepStatus bool
	// Flag indicating that comments should not be stripped.
	KeepComments bool
	// Flag indicating that output should be formatted.
	Format bool
	// The explicit working directory used to resolve relative paths.
	WorkingDirectory string
	// Flag indicating we can process directories recursively.
	RecursiveDirectories bool
	// Override the default path to the kubeconfig file.
	Kubeconfig string
	// Override the default Kubectl executor.
	KubectlExecutor func(cmd *exec.Cmd) ([]byte, error)
	// Override the default Kustomize executor.
	KustomizeExecutor func(cmd *exec.Cmd) ([]byte, error)
}

// Filter expands all of the Konjure resources using the configured executors.
func (f *Filter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	var err error

	opts := []readers.Option{
		readers.WithDefaultInputStream(f.DefaultReader),
		readers.WithWorkingDirectory(f.WorkingDirectory),
		readers.WithRecursiveDirectories(f.RecursiveDirectories),
		readers.WithKubeconfig(f.Kubeconfig),
		readers.WithKubectlExecutor(f.KubectlExecutor),
		readers.WithKustomizeExecutor(f.KustomizeExecutor),
	}

	nodes, err = (&readers.Filter{Depth: f.Depth, ReaderOptions: opts}).Filter(nodes)
	if err != nil {
		return nil, err
	}

	nodes, err = f.ResourceMetaFilter.Filter(nodes)
	if err != nil {
		return nil, err
	}

	if !f.KeepStatus {
		nodes, err = kio.FilterAll(yaml.Clear("status")).Filter(nodes)
		if err != nil {
			return nil, err
		}
	}

	if !f.KeepComments {
		nodes, err = (&kiofilters.StripCommentsFilter{}).Filter(nodes)
		if err != nil {
			return nil, err
		}
	}

	if f.Format {
		nodes, err = (&kiofilters.FormatFilter{}).Filter(nodes)
		if err != nil {
			return nil, err
		}
	}

	return nodes, nil
}
