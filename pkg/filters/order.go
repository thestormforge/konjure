package filters

import (
	"sort"

	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// InstallOrder returns a filter that sorts nodes in the order in which they
// should be created in a cluster. This uses the Helm ordering which is more
// complete than the Kustomize ordering.
func InstallOrder() kio.Filter {
	return SortByKind([]string{
		"Namespace",
		"NetworkPolicy",
		"ResourceQuota",
		"LimitRange",
		"PodSecurityPolicy",
		"PodDisruptionBudget",
		"ServiceAccount",
		"Secret",
		"SecretList",
		"ConfigMap",
		"StorageClass",
		"PersistentVolume",
		"PersistentVolumeClaim",
		"CustomResourceDefinition",
		"ClusterRole",
		"ClusterRoleList",
		"ClusterRoleBinding",
		"ClusterRoleBindingList",
		"Role",
		"RoleList",
		"RoleBinding",
		"RoleBindingList",
		"Service",
		"DaemonSet",
		"Pod",
		"ReplicationController",
		"ReplicaSet",
		"Deployment",
		"HorizontalPodAutoscaler",
		"StatefulSet",
		"Job",
		"CronJob",
		"IngressClass",
		"Ingress",
		"APIService",
	})
}

// UninstallOrder returns a filter that sorts nodes in the order in which they
// should be deleted from a cluster. This is not directly the reverse of the
// installation order.
func UninstallOrder() kio.Filter {
	return SortByKind([]string{
		"APIService",
		"Ingress",
		"IngressClass",
		"Service",
		"CronJob",
		"Job",
		"StatefulSet",
		"HorizontalPodAutoscaler",
		"Deployment",
		"ReplicaSet",
		"ReplicationController",
		"Pod",
		"DaemonSet",
		"RoleBindingList",
		"RoleBinding",
		"RoleList",
		"Role",
		"ClusterRoleBindingList",
		"ClusterRoleBinding",
		"ClusterRoleList",
		"ClusterRole",
		"CustomResourceDefinition",
		"PersistentVolumeClaim",
		"PersistentVolume",
		"StorageClass",
		"ConfigMap",
		"SecretList",
		"Secret",
		"ServiceAccount",
		"PodDisruptionBudget",
		"PodSecurityPolicy",
		"LimitRange",
		"ResourceQuota",
		"NetworkPolicy",
		"Namespace",
	})
}

// SortByKind returns a filter that sorts nodes based on the supplied list of kinds.
func SortByKind(priority []string) kio.Filter {
	order := make(map[string]int, len(priority))
	for i, p := range priority {
		order[p] = len(priority) - i
	}

	return kio.FilterFunc(func(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
		sort.SliceStable(nodes, func(i, j int) bool {
			ki, kj := nodes[i].GetKind(), nodes[j].GetKind()
			oi, oj := order[ki], order[kj]
			if oi == oj {
				return ki < kj
			}
			return oi >= oj
		})
		return nodes, nil
	})
}
