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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/pkg/gvk"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
	"sigs.k8s.io/kustomize/v3/pkg/types"
	"sigs.k8s.io/kustomize/v3/plugin/builtin"
)

// BerglasOptions is the configuration common between the generator and transformer
type BerglasOptions struct {
	GeneratorOptions *types.GeneratorOptions `json:"generatorOptions,omitempty"`
}

// BerglasGenerateOptions is the configuration for fetching secrets
type BerglasGenerateOptions struct {
	BerglasOptions `json:",inline"`
	Name           string   `json:"name"`
	References     []string `json:"refs"`
}

func NewBerglasGenerateOptions() *BerglasGenerateOptions {
	return &BerglasGenerateOptions{}
}

// BerglasTransformOptions is the configuration for modifying pod templates
type BerglasTransformOptions struct {
	BerglasOptions  `json:",inline"`
	GenerateSecrets bool `json:"-"` // TODO Change back to "generateSecrets,omitempty
}

func NewBerglasTransformOptions() *BerglasTransformOptions {
	return &BerglasTransformOptions{}
}

func (o *BerglasGenerateOptions) Run() ([]byte, error) {
	// This code uses the Kustomize code for secret generation along with the Berglas API to populate "literal" secret sources
	rf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), nil)

	// TODO Expose additional configuration options for the client
	ldr, err := NewBerglasLoader(context.Background())
	if err != nil {
		return nil, err
	}

	args := types.SecretArgs{}
	args.Name = o.Name

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

	// Generate the secret resource
	m, err := rf.FromSecretArgs(ldr, o.GeneratorOptions, args)
	if err != nil {
		return nil, err
	}

	// Add hash names (there is only one resource in the map, no need to fix references)
	p := builtin.NewHashTransformerPlugin()
	if err := p.Config(ldr, rf, nil); err != nil {
		return nil, err
	}
	if err := p.Transform(m); err != nil {
		return nil, err
	}

	return m.AsYaml()
}

func (o *BerglasTransformOptions) Run(in []byte) ([]byte, error) {
	// This code uses Kustomize code to parse and manipulate the resources
	rf := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), nil)

	// TODO Expose additional configuration options for the client
	ldr, err := NewBerglasLoader(context.Background())
	if err != nil {
		return nil, err
	}

	opts := o.GeneratorOptions
	if opts == nil && o.GenerateSecrets {
		opts = &types.GeneratorOptions{}
	} else if !o.GenerateSecrets {
		opts = nil
	}

	// Create a new mutator
	mutator := NewBerglasMutator(rf, ldr, opts)

	m, err := rf.NewResMapFromBytes(in)
	if err != nil {
		return nil, err
	}

	for _, r := range m.Resources() {
		// Mutate using the appropriate API struct
		if err := mutateResourceAs(mutator, r); err != nil {
			return nil, err
		}

		// Check if there were any new secrets that need to be added
		if err := mutator.FlushSecrets(m); err != nil {
			return nil, err
		}
	}

	// TODO What about hash names? We would need to fix name references

	return m.AsYaml()
}

// Performs the Berglas mutation on a Kustomize resource
func mutateResourceAs(m *BerglasMutator, r *resource.Resource) error {
	// Create a new typed object
	obj, err := scheme.Scheme.New(toSchemaGvk(r.GetGvk()))
	if err != nil {
		return nil // This is ignorable
	}

	// Marshal the unstructured resource to JSON and unmarshal it back into the typed structure
	if data, err := r.MarshalJSON(); err != nil {
		return err
	} else if err := json.Unmarshal(data, obj); err != nil {
		return err
	}

	// Reflectively get a pointer to the PodTemplateSpec
	template := podTemplate(obj)
	if template == nil {
		return nil
	}

	// Mutate the PodTemplateSpec and if there were changes, reverse the marshalling process back into the resource
	if didMutate, err := m.Mutate(template); err != nil {
		return err
	} else if didMutate {
		if data, err := json.Marshal(obj); err != nil {
			return err
		} else {
			return r.UnmarshalJSON(data)
		}
	}
	return nil
}

func toSchemaGvk(x gvk.Gvk) schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   x.Group,
		Version: x.Version,
		Kind:    x.Kind,
	}
}

func podTemplate(obj runtime.Object) *corev1.PodTemplateSpec {
	v := reflect.ValueOf(obj)
	if !v.CanInterface() {
		return nil
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return nil
	}

	v = v.FieldByName("Spec")
	if v.Kind() != reflect.Struct {
		return nil
	}

	v = v.FieldByName("Template")
	if !v.CanAddr() {
		return nil
	}

	v = v.Addr()
	if !v.CanInterface() {
		return nil
	}

	if t, ok := v.Interface().(*corev1.PodTemplateSpec); ok {
		return t
	}

	return nil
}
