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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type FileReader struct {
	konjurev1beta2.File
}

func (r *FileReader) Read() ([]*yaml.RNode, error) {
	var result []*yaml.RNode
	err := filepath.Walk(r.File.Path, func(path string, info os.FileInfo, err error) error {
		// Just bubble walk errors back up
		if err != nil {
			// TODO Check the annotations on the File object to see if there is any context we can add to the error
			return err
		}

		// Check to see if a directory is a Kustomize root, otherwise ignore them
		if info.IsDir() {
			if isKustomizeRoot(path) {
				n, err := konjurev1beta2.GetRNode(&konjurev1beta2.Kustomize{Root: path})
				if err != nil {
					return err
				}

				result = append(result, n)
				return filepath.SkipDir
			}

			return nil
		}

		// Try to figure out what to do based on the file extension
		switch strings.ToLower(filepath.Ext(path)) {

		case ".jsonnet":
			n, err := konjurev1beta2.GetRNode(&konjurev1beta2.Jsonnet{Filename: path})
			if err != nil {
				return err
			}

			result = append(result, n)

		case ".yaml", ".yml":
			// Just read the data in, assume it must be manifests to slurp
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			// Silently ignore errors if we cannot get any valid resources of this
			if nodes, err := kio.FromBytes(data); err == nil {
				for _, n := range nodes {
					m, err := n.GetMeta()
					if err != nil {
						continue
					}

					// Kind is required
					if m.Kind == "" {
						continue
					}

					// Name may be required
					if m.Name == "" &&
						!strings.HasSuffix(m.Kind, "List") &&
						m.APIVersion != konjurev1beta2.APIVersion {
						continue
					}

					result = append(result, n)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// isKustomizeRoot tests to see if the specified directory can be used a Kustomize root.
func isKustomizeRoot(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}

	names, err := f.Readdirnames(-1)
	_ = f.Close()
	if err != nil {
		return false
	}

	for _, n := range names {
		switch n {
		case "kustomization.yaml",
			"kustomization.yml",
			"Kustomization":
			return true
		}
	}

	return false
}
