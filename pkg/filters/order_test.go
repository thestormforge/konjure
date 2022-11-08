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
				kindAndName("APIService", "!"),
				kindAndName("Bunny", "!"),
				kindAndName("ClusterRole", "l"),
				kindAndName("ClusterRoleBinding", "s"),
				kindAndName("ClusterRoleBindingList", "t"),
				kindAndName("ClusterRoleList", "i"),
				kindAndName("ConfigMap", "f"),
				kindAndName("CronJob", "o"),
				kindAndName("CustomResourceDefinition", "i"),
				kindAndName("DaemonSet", "i"),
				kindAndName("Deployment", "d"),
				kindAndName("Fuzzy", "!"),
				kindAndName("HorizontalPodAutoscaler", "o"),
				kindAndName("Ingress", "s"),
				kindAndName("IngressClass", "u"),
				kindAndName("Job", "i"),
				kindAndName("LimitRange", "e"),
				kindAndName("Namespace", "s"),
				kindAndName("NetworkPolicy", "u"),
				kindAndName("PersistentVolume", "a"),
				kindAndName("PersistentVolumeClaim", "g"),
				kindAndName("Pod", "a"),
				kindAndName("PodDisruptionBudget", "c"),
				kindAndName("PodSecurityPolicy", "r"),
				kindAndName("ReplicaSet", "i"),
				kindAndName("ReplicationController", "l"),
				kindAndName("ResourceQuota", "p"),
				kindAndName("Role", "i"),
				kindAndName("RoleBinding", "e"),
				kindAndName("RoleBindingList", "x"),
				kindAndName("RoleList", "c"),
				kindAndName("Secret", "l"),
				kindAndName("SecretList", "i"),
				kindAndName("Service", "p"),
				kindAndName("ServiceAccount", "a"),
				kindAndName("StatefulSet", "c"),
				kindAndName("StorageClass", "r"),
			},
			expectedNames: "supercalifragilisticexpialidocious!!!",
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

func kindAndName(kind, name string) yaml.ResourceMeta {
	md := yaml.ResourceMeta{}
	md.Kind = kind
	md.Name = name
	return md
}
