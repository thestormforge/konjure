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
	"os/exec"
	"path/filepath"

	konjurev1beta2 "github.com/thestormforge/konjure/pkg/api/core/v1beta2"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type GitReader struct {
	konjurev1beta2.Git
	path string
}

func (r *GitReader) Read() ([]*yaml.RNode, error) {
	var err error
	r.path, err = ioutil.TempDir("", "konjure-git")
	if err != nil {
		return nil, err
	}

	refspec := r.Refspec
	if refspec == "" {
		refspec = "HEAD"
	}

	// Fetch the into the temporary directory
	if err := r.run("init"); err != nil {
		return nil, err
	}
	if err := r.run("remote", "add", "origin", r.Repository); err != nil {
		return nil, err
	}
	if err := r.run("fetch", "--depth=1", "origin", refspec); err != nil {
		return nil, err
	}
	if err := r.run("checkout", "FETCH_HEAD"); err != nil {
		return nil, err
	}
	if err := r.run("submodule", "update", "--init", "--recursive"); err != nil {
		return nil, err
	}

	// This creates a single File resource for the subdirectory of the Git repository
	// TODO Annotate the File resource with the Git information? (that should be whenever a Konjure resource creates another Konjure resource)
	n, err := konjurev1beta2.GetRNode(&konjurev1beta2.File{
		Path: filepath.Join(r.path, r.Context),
	})
	if err != nil {
		return nil, err
	}
	return []*yaml.RNode{n}, nil
}

func (r *GitReader) Clean() error {
	if r.path == "" {
		return nil
	}
	if err := os.RemoveAll(r.path); err != nil {
		return err
	}
	r.path = ""
	return nil
}

func (r *GitReader) run(arg ...string) error {
	cmd := exec.Command("git", arg...)
	cmd.Dir = r.path
	return cmd.Run()
}
