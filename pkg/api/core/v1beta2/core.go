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

package v1beta2

func (h *Helm) GetBin() string {
	if h.Helm.Bin == "" {
		return "helm"
	}
	return h.Helm.Bin
}

func (k *Kustomize) GetBin() string {
	return "kustomize"
}

func (k *Kubernetes) GetBin() string {
	if k.Bin == "" {
		return "kubectl"
	}
	return k.GetBin()
}

func (g *Git) GetBin() string {
	return "git"
}

func (g *Git) GetRefspec() string {
	if g.Refspec == "" {
		return "HEAD"
	}

	return g.Refspec
}
