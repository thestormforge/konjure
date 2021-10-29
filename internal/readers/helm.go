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

package readers

import (
	"fmt"
	"path/filepath"

	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"github.com/thestormforge/konjure/pkg/filters"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type HelmReader struct {
	konjurev1beta2.Helm
	Runtime

	// The path to the Helm repository cache. Corresponds to the `helm --repository-cache` option.
	RepositoryCache string
}

func (helm *HelmReader) Read() ([]*yaml.RNode, error) {
	cmd := helm.command()

	cmd.Args = append(cmd.Args, "template")

	if helm.ReleaseName != "" {
		cmd.Args = append(cmd.Args, helm.ReleaseName)
	} else {
		cmd.Args = append(cmd.Args, "--generate-name")
	}

	cmd.Args = append(cmd.Args, helm.Chart)

	if helm.Version != "" {
		cmd.Args = append(cmd.Args, "--version", helm.Version)
	}

	if helm.ReleaseNamespace != "" {
		cmd.Args = append(cmd.Args, "--namespace", helm.ReleaseNamespace)
	}

	if helm.Repository != "" {
		cmd.Args = append(cmd.Args, "--repo", helm.Repository)
	}

	for i := range helm.Values {
		switch {

		case helm.Values[i].File != "":
			// Try to expand a glob; if it fails or does not match, pass on the raw value and let Helm figure it out
			valueFiles := []string{helm.Values[i].File}
			if matches, err := filepath.Glob(helm.Values[i].File); err == nil && len(matches) > 0 {
				valueFiles = matches
			}
			for _, f := range valueFiles {
				cmd.Args = append(cmd.Args, "--values", f)
			}

		case helm.Values[i].Name != "":
			setOpt := "--set"
			if helm.Values[i].LoadFile {
				setOpt = "--set-file"
			} else if helm.Values[i].ForceString {
				setOpt = "--set-string"
			}

			cmd.Args = append(cmd.Args, setOpt, fmt.Sprintf("%s=%v", helm.Values[i].Name, helm.Values[i].Value))

		}
	}

	p := &filters.Pipeline{Inputs: []kio.Reader{cmd}}

	if !helm.IncludeTests {
		p.Filters = append(p.Filters, &filters.ResourceMetaFilter{
			AnnotationSelector: "helm.sh/hook notin (test-success, test-failure)",
		})
	}

	return p.Read()
}

func (helm *HelmReader) command() *command {
	cmd := helm.Runtime.command("helm")
	if helm.RepositoryCache != "" {
		cmd.Env = append(cmd.Env, "HELM_REPOSITORY_CACHE="+helm.RepositoryCache)
	}
	return cmd
}
