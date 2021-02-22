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
	"fmt"
	"net/http"

	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type HTTPReader struct {
	konjurev1beta2.HTTP
	Client *http.Client
}

func (r *HTTPReader) Read() ([]*yaml.RNode, error) {
	req, err := http.NewRequest(http.MethodGet, r.HTTP.URL, nil)
	if err != nil {
		return nil, err
	}

	// TODO Set Accept headers for JSON or YAML

	c := r.Client
	if c == nil {
		c = http.DefaultClient
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("invalid response code for %q: %d", r.HTTP.URL, resp.StatusCode)
	}

	// TODO Should we annotate where the (likely non-Konjure) resources originated from? Even if the default writer strips those annotations
	return (&kio.ByteReader{Reader: resp.Body}).Read()
}
