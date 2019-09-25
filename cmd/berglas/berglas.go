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

package berglas

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/GoogleCloudPlatform/berglas/pkg/berglas"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/pkg/gvk"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/types"
)

// BerglasOptions is the configuration common between the generator and transformer
type BerglasOptions struct {
	GeneratorOptions *types.GeneratorOptions `json:"generatorOptions,omitempty"`
}

// BerglasGenerateOptions is the configuration for fetching secrets
type BerglasGenerateOptions struct {
	BerglasOptions `json:",inline"`
	References     []string `json:"refs"`
}

func NewBerglasGenerateOptions() *BerglasGenerateOptions {
	return &BerglasGenerateOptions{}
}

// BerglasTransformOptions is the configuration for modifying pod templates
type BerglasTransformOptions struct {
	BerglasOptions  `json:",inline"`
	GenerateSecrets bool `json:"generateSecrets,omitempty"`
}

func NewBerglasTransformOptions() *BerglasTransformOptions {
	return &BerglasTransformOptions{}
}

func (o *BerglasGenerateOptions) Run(name string) ([]byte, error) {
	// This code uses the Kustomize code for secret generation along with the Berglas API to populate "literal" secret sources
	rmf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), nil)

	// TODO Expose additional configuration options for the client
	ldr, err := NewBerglasLoader(context.Background())
	if err != nil {
		return nil, err
	}

	args := types.SecretArgs{}
	args.Name = name

	// Add a file source for each of the configured references
	for _, ref := range o.References {
		// TODO This drops the generation from the URI fragment
		r, err := berglas.ParseReference(ref)
		if err != nil {
			return nil, err
		}
		fileSource := fmt.Sprintf("%s=%s/%s", r.Filepath(), r.Bucket(), r.Object())
		args.FileSources = append(args.FileSources, strings.TrimLeft(fileSource, "="))
	}

	// Generate the secret resource and dump it as YAML
	m, err := rmf.FromSecretArgs(ldr, o.GeneratorOptions, args)
	if err != nil {
		return nil, err
	}
	return m.AsYaml()
}

// Selectors to find resources that have a PodTemplateSpec we can mutate
var (
	ByDaemonSet   = &gvk.Gvk{Group: "apps", Kind: "DaemonSet"}
	ByDeployment  = &gvk.Gvk{Group: "apps", Kind: "Deployment"}
	ByReplicaSet  = &gvk.Gvk{Group: "apps", Kind: "ReplicaSet"}
	ByStatefulSet = &gvk.Gvk{Group: "apps", Kind: "StatefulSet"}
	ByJob         = &gvk.Gvk{Group: "batch", Kind: "Job"}
)

func (o *BerglasTransformOptions) Run(in []byte) ([]byte, error) {
	// This code uses Kustomize code to parse and manipulate the resources
	rmf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), nil)
	m, err := rmf.NewResMapFromBytes(in)
	if err != nil {
		return nil, err
	}

	ldr, err := NewBerglasLoader(context.Background())
	if err != nil {
		return nil, err
	}

	opts := o.GeneratorOptions
	if !o.GenerateSecrets {
		opts = nil
	}

	// Create a new mutator
	mutator := NewBerglasMutator(rmf, ldr, opts)

	for _, r := range m.Resources() {
		// Mutate using the appropriate API struct
		switch {
		case r.GetGvk().IsSelected(ByDaemonSet):
			if err := mutateResourceAs(mutator, r, &appsv1.DaemonSet{}); err != nil {
				return nil, err
			}
		case r.GetGvk().IsSelected(ByDeployment):
			if err := mutateResourceAs(mutator, r, &appsv1.Deployment{}); err != nil {
				return nil, err
			}
		case r.GetGvk().IsSelected(ByReplicaSet):
			if err := mutateResourceAs(mutator, r, &appsv1.ReplicaSet{}); err != nil {
				return nil, err
			}
		case r.GetGvk().IsSelected(ByStatefulSet):
			if err := mutateResourceAs(mutator, r, &appsv1.StatefulSet{}); err != nil {
				return nil, err
			}
		case r.GetGvk().IsSelected(ByJob):
			if err := mutateResourceAs(mutator, r, &batchv1.Job{}); err != nil {
				return nil, err
			}
		}

		// Check if there were any new secrets that need to be added
		if err := mutator.FlushSecrets(m); err != nil {
			return nil, err
		}
	}

	return m.AsYaml()
}

// Performs the Berglas mutation on a Kustomize resource using the supplied API binding
func mutateResourceAs(m *BerglasMutator, r *resource.Resource, v interface{}) error {
	// First marshal the resource to JSON and unmarshal it into the typed structure
	if data, err := r.MarshalJSON(); err != nil {
		return err
	} else if err := json.Unmarshal(data, v); err != nil {
		return err
	}

	// Things that have PodTemplateSpec always have them at `.spec.template`, so access that reflectively
	template := reflect.ValueOf(v).Elem().FieldByName("Spec").FieldByName("Template").Interface()
	if pts, ok := template.(corev1.PodTemplateSpec); ok {
		// Mutate the PodTemplateSpec and if there were changes, reverse the marshalling process back into the resource
		if didMutate, err := m.Mutate(&pts); err != nil {
			return err
		} else if didMutate {
			if data, err := json.Marshal(v); err != nil {
				return err
			} else {
				return r.UnmarshalJSON(data)
			}
		}
	}

	return nil
}
