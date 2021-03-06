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
	"os/exec"
	"path/filepath"

	"github.com/thestormforge/konjure/internal/filters"
	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func NewHelmReader(helm *konjurev1beta2.Helm) kio.Reader {
	helmBin := helm.Helm.Bin
	if helmBin == "" {
		helmBin = "helm"
	}
	cmd := exec.Command(helmBin)

	if helm.Helm.RepositoryCache != "" {
		cmd.Env = []string{
			"HELM_REPOSITORY_CACHE=" + helm.Helm.RepositoryCache,
		}
	}

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

	p := &Pipeline{Inputs: []kio.Reader{(*ExecReader)(cmd)}}

	if !helm.IncludeTests {
		p.Filters = append(p.Filters, &filters.SelectorFilter{
			AnnotationSelector: "helm.sh/hook in (test-success, test-failure)",
			Negate:             true,
		})
	}

	return p
}
