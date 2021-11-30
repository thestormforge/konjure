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
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/kio/kioutil"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type FileReader struct {
	konjurev1beta2.File

	// Flag indicating we are allowed to recurse into directories.
	Recurse bool
	// Function used to determine an absolute path.
	Abs func(path string) (string, error)
}

func (r *FileReader) Read() ([]*yaml.RNode, error) {
	root, err := r.root()
	if err != nil {
		return nil, err
	}

	var result []*yaml.RNode
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// Just bubble walk errors back up
		if err != nil {
			// TODO Check the annotations on the File object to see if there is any context we can add to the error
			return err
		}

		if info.IsDir() {
			// Determine if we are allowed to recurse into the directory
			if !r.Recurse && path != root {
				return filepath.SkipDir
			}

			// Check to see if a directory is a Kustomize root
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

		case ".json", ".yaml", ".yml", "":
			// Just read the data in, assume it must be manifests to slurp
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			br := &kio.ByteReader{
				Reader: bytes.NewReader(data),
				SetAnnotations: map[string]string{
					kioutil.PathAnnotation: path,
				},
			}

			nodes, err := br.Read()
			if err != nil {
				return err
			}

			// Only keep things that appear to be Kube resources
			for _, n := range nodes {
				if keepNode(n) {
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

// root returns the root path, failing if it cannot be made into an absolute path.
func (r *FileReader) root() (path string, err error) {
	path = r.Path

	// If available, use the path resolver
	if r.Abs != nil {
		path, err = r.Abs(path)
		if err != nil {
			return "", err
		}
	}

	// Even if we used Abs, make sure the result is actually absolute
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("unable to resolve relative path %s", r.Path)
	}

	return path, nil
}

// keepNode tests the supplied node to see if it should be included in the result.
func keepNode(node *yaml.RNode) bool {
	m, err := node.GetMeta()
	if err != nil {
		return false
	}

	// Kind is required
	if m.Kind == "" {
		return false
	}

	switch {

	case m.APIVersion == konjurev1beta2.APIVersion:
		// Keep all Konjure resources
		return true

	case strings.HasPrefix(m.APIVersion, "kustomize.config.k8s.io/"):
		// Keep all Kustomize resources (special case when the Kustomization wasn't expanded)
		return true

	case strings.HasSuffix(m.Kind, "List"):
		// Keep list resources
		return true

	default:
		// Keep other resources only if they have a name
		return m.Name != ""
	}
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
