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

package spec

import (
	"fmt"

	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
)

type Formatter struct {
}

func (f *Formatter) Encode(obj any) (string, error) {
	switch s := obj.(type) {
	case *konjurev1beta2.Resource:
		if len(s.Resources) == 1 {
			return s.Resources[0], nil
		}

	case *konjurev1beta2.Helm:
		// TODO Should we attempt to make this into a URL?

	case *konjurev1beta2.Jsonnet:
		if s.Filename != "" &&
			s.Code == "" &&
			len(s.JsonnetPath) == 0 &&
			len(s.ExternalVariables) == 0 &&
			len(s.TopLevelArguments) == 0 &&
			s.JsonnetBundlerPackageHome == "" &&
			!s.JsonnetBundlerRefresh {
			return s.Filename, nil
		}

	case *konjurev1beta2.Kubernetes:
		// Do nothing

	case *konjurev1beta2.Kustomize:
		return s.Root, nil

	case *konjurev1beta2.Secret:
		// There is no specification form for secrets

	case *konjurev1beta2.Git:
		// TODO This is probably more complex because of all the allowed formats

	case *konjurev1beta2.HTTP:
		return s.URL, nil

	case *konjurev1beta2.File:
		return s.Path, nil
	}

	return "", fmt.Errorf("object cannot be formatted")
}
