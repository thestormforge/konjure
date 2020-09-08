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

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/api/resid"
	"sigs.k8s.io/kustomize/api/types"
)

func Test_createIfNotPresent(t *testing.T) {
	cases := []struct {
		desc     string
		group    string
		version  string
		fs       *types.FieldSpec
		expected bool
	}{
		// ReplicationController

		{
			desc:     "v1.ReplicationController",
			version:  "v1",
			fs:       fieldSpec("spec/selector", true, "", "v1", "ReplicationController"),
			expected: false,
		},

		// Deployment

		{
			desc:     "apps.v1.Deployment",
			group:    "apps",
			version:  "v1",
			fs:       fieldSpec("spec/selector/matchLabels", true, "", "", "Deployment"),
			expected: true,
		},
		{
			desc:     "apps.v1beta2.Deployment",
			group:    "apps",
			version:  "v1beta2",
			fs:       fieldSpec("spec/selector/matchLabels", true, "", "", "Deployment"),
			expected: true,
		},
		{
			desc:     "apps.v1beta1.Deployment",
			group:    "apps",
			version:  "v1beta1",
			fs:       fieldSpec("spec/selector/matchLabels", true, "", "", "Deployment"),
			expected: false,
		},
		{
			desc:     "extensions.v1beta1.Deployment",
			group:    "extensions",
			version:  "v1beta1",
			fs:       fieldSpec("spec/selector/matchLabels", true, "", "", "Deployment"),
			expected: false,
		},

		// ReplicaSet

		{
			desc:     "apps.v1.ReplicaSet",
			group:    "apps",
			version:  "v1",
			fs:       fieldSpec("spec/selector/matchLabels", true, "", "", "ReplicaSet"),
			expected: true,
		},
		{
			desc:     "apps.v1beta2.ReplicaSet",
			group:    "apps",
			version:  "v1beta2",
			fs:       fieldSpec("spec/selector/matchLabels", true, "", "", "ReplicaSet"),
			expected: true,
		},
		{
			desc:     "extensions.v1beta1.ReplicaSet",
			group:    "extensions",
			version:  "v1beta1",
			fs:       fieldSpec("spec/selector/matchLabels", true, "", "", "ReplicaSet"),
			expected: false,
		},

		// DaemonSet

		{
			desc:     "apps.v1.DaemonSet",
			group:    "apps",
			version:  "v1",
			fs:       fieldSpec("spec/selector/matchLabels", true, "", "", "DaemonSet"),
			expected: true,
		},
		{
			desc:     "apps.v1beta2.DaemonSet",
			group:    "apps",
			version:  "v1beta2",
			fs:       fieldSpec("spec/selector/matchLabels", true, "", "", "DaemonSet"),
			expected: true,
		},
		{
			desc:     "extensions.v1beta1.DaemonSet",
			group:    "extensions",
			version:  "v1beta1",
			fs:       fieldSpec("spec/selector/matchLabels", true, "", "", "DaemonSet"),
			expected: false,
		},

		// StatefulSet

		{
			desc:     "apps.v1.StatefulSet",
			group:    "apps",
			version:  "v1",
			fs:       fieldSpec("spec/selector/matchLabels", true, "", "", "StatefulSet"),
			expected: true,
		},
		{
			desc:     "apps.v1beta2.StatefulSet",
			group:    "apps",
			version:  "v1beta2",
			fs:       fieldSpec("spec/selector/matchLabels", true, "", "", "StatefulSet"),
			expected: true,
		},
		{
			desc:     "apps.v1beta1.StatefulSet",
			group:    "apps",
			version:  "v1beta1",
			fs:       fieldSpec("spec/selector/matchLabels", true, "", "", "StatefulSet"),
			expected: false,
		},

		// These tests just confirm we aren't messing with stuff we shouldn't be

		{
			desc:     "apps.v1beta2 labels",
			group:    "apps",
			version:  "v1beta2",
			fs:       fieldSpec("metadata/labels", true, "", "", ""),
			expected: true,
		},
		{
			desc:     "extensions.v1beta1 labels",
			group:    "extensions",
			version:  "v1beta1",
			fs:       fieldSpec("metadata/labels", true, "", "", ""),
			expected: true,
		},
		{
			desc:     "batch.v1.Job selector",
			group:    "batch",
			version:  "v1",
			fs:       fieldSpec("spec/selector/matchLabels", false, "batch", "", "Job"),
			expected: false,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert.Equal(t, c.expected, createIfNotPresent(c.group, c.version, c.fs))
		})
	}
}

func fieldSpec(path string, create bool, group, version, kind string) *types.FieldSpec {
	return &types.FieldSpec{
		Gvk: resid.Gvk{
			Group:   group,
			Version: version,
			Kind:    kind,
		},
		Path:               path,
		CreateIfNotPresent: create,
	}
}
