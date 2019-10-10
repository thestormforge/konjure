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
	Source string
	DryRun bool

	groups  []string
	plugins []metav1.GroupVersionKind
}

func NewInitializeOptions() *InitializeOptions {
	return &InitializeOptions{}
}

func (o *InitializeOptions) Complete() error {
	var err error
	g := make(map[string]bool, len(o.plugins))

	if o.Source == "" {
		if o.Source, err = os.Executable(); err != nil {
			return err
		}
	}

	for _, c := range NewKustomizeCommand().Commands() {
		gvk := util.ExecPluginGVK(c)
		if gvk != nil {
			o.plugins = append(o.plugins, *gvk)
			g[gvk.Group] = true
		}
	}

	for k := range g {
		o.groups = append(o.groups, k)
	}

	return nil
}

func (o *InitializeOptions) Run(out io.Writer) error {
	// Early dismissal
	if len(o.plugins) == 0 {
		return nil
	}

	if o.DryRun {
		return o.checkLinks(out)
	}
	return o.createLinks(out)
}

// TODO Instead of taking a writer, we should return a map of Kind -> status or use a logger or something...

func (o *InitializeOptions) createLinks(out io.Writer) error {
	// TODO Init should clean out old versions as well
	// Create a symlink for each plugin
	pluginDir := util.PluginDirectory()
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

func (o *InitializeOptions) checkLinks(out io.Writer) error {
	// Find what symlinks are installed
	pluginDir := util.PluginDirectory()
	installed := findInstalled(pluginDir, o.groups)
	for _, gvk := range o.plugins {
		status := "OK"
		link := filepath.Join(pluginDir, gvk.Group, gvk.Version, strings.ToLower(gvk.Kind), gvk.Kind)
		if _, err := os.Lstat(link); err != nil {
			// It does not exist
			status = "Missing"
			for _, i := range installed {
				if gvk.Group == i.Group && gvk.Kind == i.Kind {
					status = "Wrong version (" + i.Version + ")"
					break
				}
			}
		} else if l, err := os.Readlink(link); err == nil {
			// If it exists make sure it is pointing to the right place
			if l != o.Source {
				status = "Incorrect link (" + l + ")"
			}
		}
		_, _ = fmt.Fprintf(out, "%s ... %s\n", gvk.Kind, status)
	}

	return nil
}

// Finds all of the installed plugins under the specified directory and groups
func findInstalled(pluginDir string, groups []string) []metav1.GroupVersionKind {
	var installed []metav1.GroupVersionKind
	for _, g := range groups {
		_ = filepath.Walk(filepath.Join(pluginDir, g), func(path string, info os.FileInfo, err error) error {
			rel, err := filepath.Rel(pluginDir, path)
			if err != nil {
				return err
			}

			gvk := metav1.GroupVersionKind{}
			rel, gvk.Kind = filepath.Split(filepath.Clean(rel))
			rel, lowerKind := filepath.Split(filepath.Clean(rel))
			rel, gvk.Version = filepath.Split(filepath.Clean(rel))
			gvk.Group = filepath.Clean(rel)

			if lowerKind == strings.ToLower(gvk.Kind) {
				installed = append(installed, gvk)
			}
			return nil
		})
	}
	return installed
}
