/*
Copyright 2019 GramLabs, Inc.

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

package transformer

import (
	"testing"

	. "github.com/onsi/gomega"
	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/types"
)

func Test_createIfNotPresent(t *testing.T) {
	g := NewGomegaWithT(t)

	// In all these cases the "fs" comes from the default built-in configuration
	// The following configuration was used at the time this test was written:
	// https://raw.githubusercontent.com/kubernetes-sigs/kustomize/077c7b2d20bfdb0de78e6873a4ae1ce08afa1c40/api/konfig/builtinpluginconsts/commonlabels.go
	var fs *types.FieldSpec

	fs = fieldSpec("spec/selector", true, "", "v1", "ReplicationController")
	g.Expect(createIfNotPresent(x("", "v1", "ReplicationController"), fs)).To(Equal(false))

	fs = fieldSpec("spec/selector/matchLabels", true, "", "", "Deployment")
	g.Expect(createIfNotPresent(x("apps", "v1", "Deployment"), fs)).To(Equal(true))
	g.Expect(createIfNotPresent(x("apps", "v1beta2", "Deployment"), fs)).To(Equal(true))
	g.Expect(createIfNotPresent(x("apps", "v1beta1", "Deployment"), fs)).To(Equal(false))
	g.Expect(createIfNotPresent(x("extensions", "v1beta1", "Deployment"), fs)).To(Equal(false))

	fs = fieldSpec("spec/selector/matchLabels", true, "", "", "ReplicaSet")
	g.Expect(createIfNotPresent(x("apps", "v1", "ReplicaSet"), fs)).To(Equal(true))
	g.Expect(createIfNotPresent(x("apps", "v1beta2", "ReplicaSet"), fs)).To(Equal(true))
	g.Expect(createIfNotPresent(x("extensions", "v1beta1", "ReplicaSet"), fs)).To(Equal(false))

	fs = fieldSpec("spec/selector/matchLabels", true, "", "", "DaemonSet")
	g.Expect(createIfNotPresent(x("apps", "v1", "DaemonSet"), fs)).To(Equal(true))
	g.Expect(createIfNotPresent(x("apps", "v1beta2", "DaemonSet"), fs)).To(Equal(true))
	g.Expect(createIfNotPresent(x("extensions", "v1beta1", "DaemonSet"), fs)).To(Equal(false))

	fs = fieldSpec("spec/selector/matchLabels", true, "apps", "", "StatefulSet")
	g.Expect(createIfNotPresent(x("apps", "v1", "StatefulSet"), fs)).To(Equal(true))
	g.Expect(createIfNotPresent(x("apps", "v1beta2", "StatefulSet"), fs)).To(Equal(true))
	g.Expect(createIfNotPresent(x("apps", "v1beta1", "StatefulSet"), fs)).To(Equal(false))

	// These tests just confirm we aren't messing with stuff we shouldn't be

	fs = fieldSpec("metadata/labels", true, "", "", "")
	g.Expect(createIfNotPresent(x("apps", "v1beta2", "Deployment"), fs)).To(Equal(true))
	g.Expect(createIfNotPresent(x("extensions", "v1beta1", "Deployment"), fs)).To(Equal(true))

	fs = fieldSpec("spec/selector/matchLabels", false, "batch", "", "Job")
	g.Expect(createIfNotPresent(x("batch", "v1", "Job"), fs)).To(Equal(false))
}

func x(group, version, kind string) resid.Gvk {
	return resid.Gvk{
		Group:   group,
		Version: version,
		Kind:    kind,
	}
}

func fieldSpec(path string, create bool, group, version, kind string) *types.FieldSpec {
	return &types.FieldSpec{
		Gvk:                x(group, version, kind),
		Path:               path,
		CreateIfNotPresent: create,
	}
}
