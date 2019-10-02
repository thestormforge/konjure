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

package kustomize

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/carbonrelay/konjure/cmd/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InitializeOptions struct {
	Source  string
	plugins []metav1.GroupVersionKind
}

func NewInitializeOptions() *InitializeOptions {
	return &InitializeOptions{}
}

func (o *InitializeOptions) Complete() error {
	var err error

	if o.Source == "" {
		if o.Source, err = os.Executable(); err != nil {
			return err
		}
	}

	for _, c := range NewKustomizeCommand().Commands() {
		gvk := util.ExecPluginGVK(c)
		if gvk != nil {
			o.plugins = append(o.plugins, *gvk)
		}
	}

	return nil
}

func (o *InitializeOptions) Run(out io.Writer) error {
	// Early dismissal
	if len(o.plugins) == 0 {
		return nil
	}

	// This is not a full XDG Base Directory implementation, just enough for Kustomize
	// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		// NOTE: This can produce just ".config" if the environment variable isn't set
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	pluginDir := filepath.Join(configDir, "kustomize", "plugin")

	// Create a symlink for each plugin
	for _, gvk := range o.plugins {
		// Ensure the directory exists
		dir := filepath.Join(pluginDir, gvk.Group, gvk.Version, strings.ToLower(gvk.Kind))
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}

		// Remove the existing link
		link := filepath.Join(dir, gvk.Kind)
		if _, err := os.Lstat(link); err == nil {
			if err = os.Remove(link); err != nil {
				return err
			}
		}

		// Link the executable
		if err := os.Symlink(o.Source, link); err != nil {
			return err
		} else {
			_, _ = fmt.Fprintf(out, "Created %s\n", link)
		}
	}

	return nil
}
