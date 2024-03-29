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
	"fmt"
	"path"
	"strings"

	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"github.com/thestormforge/konjure/pkg/filters"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type KubernetesReader struct {
	konjurev1beta2.Kubernetes
	Runtime

	// Override the default path to the kubeconfig file.
	Kubeconfig string
	// Override the default kubeconfig context.
	Context string
	// The list of default types to use if none are specified.
	DefaultTypes []string
}

func (k *KubernetesReader) Read() ([]*yaml.RNode, error) {
	p := &filters.Pipeline{}

	var namespaces []string
	if k.AllNamespaces {
		namespaces = []string{""}
	} else if ns, err := k.namespaces(); err != nil {
		return nil, err
	} else {
		namespaces = ns
	}

	types, err := k.types()
	if err != nil {
		return nil, err
	}

	for _, ns := range namespaces {
		cmd := k.command()
		cmd.Args = append(cmd.Args, "get")
		cmd.Args = append(cmd.Args, "--ignore-not-found")
		cmd.Args = append(cmd.Args, "--output", "yaml")
		cmd.Args = append(cmd.Args, "--selector", k.Selector)
		cmd.Args = append(cmd.Args, "--field-selector", k.FieldSelector)

		if k.AllNamespaces {
			cmd.Args = append(cmd.Args, "--all-namespaces")
		}
		if ns != "" {
			cmd.Args = append(cmd.Args, "--namespace", ns)
		}

		cmd.Args = append(cmd.Args, strings.Join(types, ","))

		p.Inputs = append(p.Inputs, cmd)
	}

	return p.Read()
}

func (k *KubernetesReader) command() *command {
	cmd := k.Runtime.command("kubectl")
	if k.Kubeconfig != "" {
		cmd.Args = append(cmd.Args, "--kubeconfig", k.Kubeconfig)
	}
	if k.Context != "" {
		cmd.Args = append(cmd.Args, "--context", k.Context)
	}
	return cmd
}

func (k *KubernetesReader) namespaces() ([]string, error) {
	if k.Namespace != "" {
		return []string{k.Namespace}, nil
	}

	if len(k.Namespaces) > 0 {
		return k.Namespaces, nil
	}

	if k.NamespaceSelector == "" {
		return []string{""}, nil
	}

	cmd := k.command()
	cmd.Args = append(cmd.Args, "get")
	cmd.Args = append(cmd.Args, "namespace")
	cmd.Args = append(cmd.Args, "--selector", k.NamespaceSelector)
	cmd.Args = append(cmd.Args, "--output", "name")
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

func (k *KubernetesReader) types() ([]string, error) {
	var types []string
	for _, t := range k.Types {
		if t != "" {
			types = append(types, t)
		}
	}
	if len(types) > 0 {
		return types, nil
	}

	for _, t := range k.DefaultTypes {
		if t != "" {
			types = append(types, t)
		}
	}

	if len(types) > 0 {
		return types, nil
	}

	return nil, fmt.Errorf("no types specified")
}
