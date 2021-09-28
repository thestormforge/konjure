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
	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type KustomizeReader struct {
	konjurev1beta2.Kustomize
	Runtime
}

func (kustomize *KustomizeReader) Read() ([]*yaml.RNode, error) {
	cmd := kustomize.command()
	cmd.Args = append(cmd.Args, "build", kustomize.Root)
	return cmd.Read()
}

func (kustomize *KustomizeReader) command() *command {
	cmd := kustomize.Runtime.command("kustomize")
	return cmd
}
