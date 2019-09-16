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

package initialize

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/carbonrelay/konjure/cmd/kustomize"
)

type InitializeOptions struct {
	Source  string
	Plugins []string
	DryRun  bool
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

	for _, c := range kustomize.NewKustomizeCommand().Commands() {
		o.Plugins = append(o.Plugins, c.Name())
	}

	return nil
}

func (o *InitializeOptions) Run(out io.Writer) error {
	// Early dismissal
	if len(o.Plugins) == 0 {
		return nil
	}

	// This is not a full XDG Base Directory implementation, just enough for Kustomize
	// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		// NOTE: This can produce just ".config" if the environment variable isn't set
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	pluginDir := filepath.Join(configDir, "kustomize", "plugin", path.Join("konjure.carbonrelay.com", "v1"))

	// Create a symlink for each
	for _, kind := range o.Plugins {
		// Ensure the directory exists
		dir := filepath.Join(pluginDir, strings.ToLower(kind))
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}

		// Remove the existing link
		link := filepath.Join(dir, kind)
		if _, err := os.Lstat(link); err == nil {
			if err = os.Remove(link); err != nil {
				return err
			}
		}

		// Link the executable
		if o.DryRun {
			_, _ = fmt.Fprintf(out, "ln -s %s %s\n", o.Source, link)
		} else if err := os.Symlink(o.Source, link); err != nil {
			return err
		}
	}

	return nil
}
