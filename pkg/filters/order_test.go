/*
Copyright 2022 CloudBolt, Inc.

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

package filters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestSortByKind(t *testing.T) {
	cases := []struct {
		desc          string
		sort          kio.Filter
		resources     []yaml.ResourceMeta
		expectedNames string
	}{
		{
			desc: "install order",
			sort: InstallOrder(),
			resources: []yaml.ResourceMeta{
				{Kind: "APIService", Name: "!"},
				{Kind: "Bunny", Name: "!"},
				{Kind: "ClusterRole", Name: "l"},
				{Kind: "ClusterRoleBinding", Name: "s"},
				{Kind: "ClusterRoleBindingList", Name: "t"},
				{Kind: "ClusterRoleList", Name: "i"},
				{Kind: "ConfigMap", Name: "f"},
				{Kind: "CronJob", Name: "o"},
				{Kind: "CustomResourceDefinition", Name: "i"},
				{Kind: "DaemonSet", Name: "i"},
				{Kind: "Deployment", Name: "d"},
				{Kind: "Fuzzy", Name: "!"},
				{Kind: "HorizontalPodAutoscaler", Name: "o"},
				{Kind: "Ingress", Name: "s"},
				{Kind: "IngressClass", Name: "u"},
				{Kind: "Job", Name: "i"},
				{Kind: "LimitRange", Name: "e"},
				{Kind: "Namespace", Name: "s"},
				{Kind: "NetworkPolicy", Name: "u"},
				{Kind: "PersistentVolume", Name: "a"},
				{Kind: "PersistentVolumeClaim", Name: "g"},
				{Kind: "Pod", Name: "a"},
				{Kind: "PodDisruptionBudget", Name: "c"},
				{Kind: "PodSecurityPolicy", Name: "r"},
				{Kind: "ReplicaSet", Name: "i"},
				{Kind: "ReplicationController", Name: "l"},
				{Kind: "ResourceQuota", Name: "p"},
				{Kind: "Role", Name: "i"},
				{Kind: "RoleBinding", Name: "e"},
				{Kind: "RoleBindingList", Name: "x"},
				{Kind: "RoleList", Name: "c"},
				{Kind: "Secret", Name: "l"},
				{Kind: "SecretList", Name: "i"},
				{Kind: "Service", Name: "p"},
				{Kind: "ServiceAccount", Name: "a"},
				{Kind: "StatefulSet", Name: "c"},
				{Kind: "StorageClass", Name: "r"},
			},
			expectedNames: "supercalifragilisticexpialidocious!!!",
		},
		{
			desc: "tiebreaker",
			sort: InstallOrder(),
			resources: []yaml.ResourceMeta{
				{Kind: "World", Name: "france-2"},
				{Kind: "Cup22", Name: "argentina-4"},
			},
			expectedNames: "argentina-4france-2",
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			nodes := make([]*yaml.RNode, 0, len(tc.resources))
			for _, md := range tc.resources {
				v := yaml.Node{}
				if err := v.Encode(&md); assert.NoError(t, err) {
					nodes = append(nodes, yaml.NewRNode(&v))
				}
			}

			actualNodes, err := tc.sort.Filter(nodes)
			if assert.NoError(t, err) {
				actualNames := ""
				for _, n := range actualNodes {
					actualNames += n.GetName()
				}
				assert.Equal(t, tc.expectedNames, actualNames)
			}
		})
	}
}
