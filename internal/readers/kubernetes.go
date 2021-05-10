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
	"bufio"
	"bytes"
	"os/exec"
	"path"
	"strings"

	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

func NewKubernetesReader(k *konjurev1beta2.Kubernetes) kio.Reader {
	p := &Pipeline{}

	namespaces, err := namespaces(k)
	if err != nil {
		return &ErrorReader{error: err}
	}

	for _, ns := range namespaces {
		kubectlBin := k.Bin
		if kubectlBin == "" {
			kubectlBin = "kubectl"
		}
		cmd := exec.Command(kubectlBin)

		if k.Kubeconfig != "" {
			cmd.Args = append(cmd.Args, "--kubeconfig", k.Kubeconfig)
		}
		if k.Context != "" {
			cmd.Args = append(cmd.Args, "--context", k.Context)
		}
		if ns != "" {
			cmd.Args = append(cmd.Args, "--namespace", ns)
		}

		cmd.Args = append(cmd.Args, "get")
		cmd.Args = append(cmd.Args, "--ignore-not-found")
		cmd.Args = append(cmd.Args, "--output", "yaml")
		cmd.Args = append(cmd.Args, "--selector", k.Selector)
		if len(k.Types) > 0 {
			cmd.Args = append(cmd.Args, strings.Join(k.Types, ","))
		} else {
			cmd.Args = append(cmd.Args, "deployments,statefulsets,configmaps")
		}

		p.Inputs = append(p.Inputs, (*ExecReader)(cmd))
	}

	return p
}

func namespaces(k *konjurev1beta2.Kubernetes) ([]string, error) {
	if k.Namespace != "" {
		return []string{k.Namespace}, nil
	}

	if len(k.Namespaces) > 0 {
		return k.Namespaces, nil
	}

	if k.NamespaceSelector == "" {
		return []string{""}, nil
	}

	name := k.Bin
	if name == "" {
		name = "kubectl"
	}

	cmd := exec.Command(name, "get", "namespace", "--selector", k.NamespaceSelector, "--output", "name")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var namespaces []string
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		namespaces = append(namespaces, path.Base(scanner.Text()))
	}

	return namespaces, nil
}
