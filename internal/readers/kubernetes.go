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
		return &ErrorReader{err: err}
	}

	for _, ns := range namespaces {
		r := &ExecReader{Name: k.Bin}
		if r.Name == "" {
			r.Name = "kubectl"
		}
		if k.Kubeconfig != "" {
			r.Args = append(r.Args, "--kubeconfig", k.Kubeconfig)
		}
		if k.Context != "" {
			r.Args = append(r.Args, "--context", k.Context)
		}
		if ns != "" {
			r.Args = append(r.Args, "--namespace", ns)
		}

		r.Args = append(r.Args, "get")
		r.Args = append(r.Args, "--ignore-not-found")
		r.Args = append(r.Args, "--output", "yaml")
		r.Args = append(r.Args, "--selector", k.LabelSelector)
		if len(k.Types) > 0 {
			r.Args = append(r.Args, strings.Join(k.Types, ","))
		} else {
			r.Args = append(r.Args, "deployments,statefulsets,configmaps")
		}

		p.Inputs = append(p.Inputs, r)
	}

	return p
}

func namespaces(k *konjurev1beta2.Kubernetes) ([]string, error) {
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
