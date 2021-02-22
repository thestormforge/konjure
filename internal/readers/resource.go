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
	"io"

	"github.com/thestormforge/konjure/internal/spec"
	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ResourceReader generates Konjure resource nodes by parsing the configured resource specifications.
type ResourceReader struct {
	// The list of resource specifications to generate Konjure resources from.
	Resources []string
	// The byte stream of (possibly non-Konjure) resources to read give the
	// resource specification of "-". Generally this should just be `os.Stdin`.
	Reader io.Reader
}

// Read produces parses resource specifications and returns resource nodes.
func (r *ResourceReader) Read() ([]*yaml.RNode, error) {
	result := kio.ResourceNodeSlice{}

	parser := spec.Parser{Reader: r.Reader}
	for _, res := range r.Resources {
		// Parse the resource specification and append the result
		res, err := parser.Decode(res)
		if err != nil {
			return nil, err
		}

		if res == nil {
			continue
		}

		switch rn := res.(type) {

		case kio.Reader:
			// It is possible the spec parser returns a reader directly (e.g. when reading stdin)
			ns, err := rn.Read()
			if err != nil {
				return nil, err
			}

			result = append(result, ns...)

		default:
			// Assume the spec resulted in a Konjure type: GetRNode will fail if it did not
			n, err := konjurev1beta2.GetRNode(rn)
			if err != nil {
				return nil, err
			}

			result = append(result, n)
		}
	}

	return result, nil
}
