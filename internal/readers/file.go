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
	"io/fs"
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

			// See if the directory itself expands
			if result, err = r.readDir(result, path); err != nil {
				return err
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
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			br := &kio.ByteReader{
				Reader: bytes.NewReader(data),
				SetAnnotations: map[string]string{
					kioutil.PathAnnotation: path,
				},
			}

			// Try to parse the file
			nodes, err := br.Read()
			if err != nil && filepath.Ext(path) != "" {
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

// readDir attempts to expand a directory into Konjure nodes. If any nodes are returned, traversal of the directory
// is skipped.
func (r *FileReader) readDir(result []*yaml.RNode, path string) ([]*yaml.RNode, error) {
	// Read the directory listing, ignore errors
	d, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	dirContents, _ := d.Readdirnames(-1)
	_ = d.Close()

	// This is a Git repository, but since it is already cloned, we can just skip it
	if filepath.Base(path) == ".git" && containsAll(dirContents, "objects", "refs", "HEAD") { // Just sanity check the dir contents
		return result, fs.SkipDir
	}

	// Look for directory contents that indicate we should handle this specially
	for _, name := range dirContents {
		switch name {
		case "kustomization.yaml",
			"kustomization.yml",
			"Kustomization":
			// The path is a Kustomization root: return a Kustomize resource, so it gets expanded correctly
			n, err := konjurev1beta2.GetRNode(&konjurev1beta2.Kustomize{Root: path})
			if err != nil {
				return nil, err
			}

			result = append(result, n)
			return result, fs.SkipDir

		case "Chart.yaml":
			// The path is a Helm chart: return a Helm resource, so we don't fail parsing YAML templates
			// TODO Read the chart...
			return result, fs.SkipDir
		}
	}

	return result, nil
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

// containsAll checks that all values are present.
func containsAll(haystack []string, needles ...string) bool {
	for i := range haystack {
		for j := range needles {
			if haystack[i] == needles[j] {
				needles = append(needles[:j], needles[j+1:]...)
				break
			}
		}
	}
	return len(needles) == 0
}
