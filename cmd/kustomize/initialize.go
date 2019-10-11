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

type Plugin struct {
	metav1.GroupVersionKind
	Supported         bool
	InstalledVersions []string
}

type InitializeOptions struct {
	PluginDir string
	Source    string
	Kinds     []string
	DryRun    bool
}

func NewInitializeOptions() *InitializeOptions {
	return &InitializeOptions{}
}

func (o *InitializeOptions) Complete() error {
	// Determine the directory where plugins are located
	if o.PluginDir == "" {
		o.PluginDir = util.PluginDirectory()
	}

	// Determine the source to use for the symlinks
	if o.Source == "" {
		var err error
		if o.Source, err = os.Executable(); err != nil {
			return err
		}
	}

	return nil
}

func (o *InitializeOptions) Run(out io.Writer) error {
	// Load and filter the plugin list
	plugins := LoadPlugins(o.PluginDir)
	if len(o.Kinds) > 0 {
		plugins = FilterPlugins(plugins, o.Kinds)
	}

	// Generate the status map
	var status map[string]string
	var err error
	if o.DryRun {
		status, err = o.checkLinks(plugins, out)
	} else {
		status, err = o.createLinks(plugins)
	}
	if err != nil {
		return err
	}

	// Print it out
	ok := true
	for k, v := range status {
		_, _ = fmt.Fprintf(out, "%s ... %s\n", k, v)

		if v != "OK" {
			ok = false
		}
	}

	// Only return an error for "not OK" if this was a dry run
	if o.DryRun && !ok {
		return fmt.Errorf("plugin not OK")
	}

	return nil
}

func (o *InitializeOptions) createLinks(plugins []Plugin) (map[string]string, error) {
	status := make(map[string]string, len(plugins))
	for i := range plugins {
		p := &plugins[i]
		status[p.Kind] = "OK"

		// Remove the enclosing directory for unsupported plugins, create it for everyone else
		dir := filepath.Dir(util.ExecPluginPath(o.PluginDir, &p.GroupVersionKind))
		if !p.Supported {
			if err := os.RemoveAll(dir); err != nil {
				return nil, err
			}
			status[p.Kind] = "Removed"
			continue
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}

		// Re-link all of the installed/supported versions
		for _, path := range pluginPaths(o.PluginDir, p) {
			if _, err := os.Lstat(path); err == nil {
				if l, err := os.Readlink(path); err != nil {
					return nil, err
				} else if l != o.Source {
					if err = os.Remove(path); err != nil {
						return nil, err
					}
					status[p.Kind] = "Updated"
				} else {
					continue
				}
			} else {
				status[p.Kind] = "Created"
			}

			if err := os.Symlink(o.Source, path); err != nil {
				return nil, err
			}
		}
	}
	return status, nil
}

func (o *InitializeOptions) checkLinks(plugins []Plugin, out io.Writer) (map[string]string, error) {
	status := make(map[string]string, len(plugins))
	for i := range plugins {
		p := &plugins[i]
		status[p.Kind] = "OK"

		if !p.Supported {
			status[p.Kind] = "Unsupported"
			continue
		}
		if !p.installed() {
			status[p.Kind] = "Missing"
			continue
		}

		for _, path := range pluginPaths(o.PluginDir, p) {
			if l, err := os.Readlink(path); err != nil {
				return nil, err
			} else {
				if l != o.Source {
					status[p.Kind] = "Needs Update"
					break
				}
			}
		}
	}
	return status, nil
}

func FilterPlugins(plugins []Plugin, filters []string) []Plugin {
	var filtered []Plugin
	for i := range plugins {
		for _, f := range filters {
			// TODO Is github.com/gobwas/glob is a transitive dependency already?
			if ok, err := filepath.Match(f, plugins[i].Kind); err == nil && ok {
				filtered = append(filtered, plugins[i])
				break
			}
		}
	}
	return filtered
}

func LoadPlugins(pluginDir string) []Plugin {
	var plugins []Plugin
	groups := make(map[string]bool)

	// First load the currently supported plugins
	for _, c := range NewKustomizeCommand().Commands() {
		gvk := util.ExecPluginGVK(c)
		if gvk != nil {
			groups[gvk.Group] = true
			plugins = append(plugins, Plugin{GroupVersionKind: *gvk, Supported: true})
		}
	}

	// Next, merge in the installed plugins
	for _, gvk := range findInstalledPlugins(pluginDir, groups) {
		p := findCompatiblePlugin(plugins, &gvk)
		if p == nil {
			p = &Plugin{GroupVersionKind: gvk}
			plugins = append(plugins, *p)
		}
		p.InstalledVersions = append(p.InstalledVersions, gvk.Version)
	}

	return plugins
}

// Finds a plugin from the supplied list with the same group and kind
func findCompatiblePlugin(plugins []Plugin, gvk *metav1.GroupVersionKind) *Plugin {
	for i := range plugins {
		if plugins[i].Group == gvk.Group && plugins[i].Kind == gvk.Kind {
			return &plugins[i]
		}
	}
	return nil
}

// Finds all of the installed plugins under the specified directory and groups
func findInstalledPlugins(pluginDir string, groups map[string]bool) []metav1.GroupVersionKind {
	var installed []metav1.GroupVersionKind
	for g := range groups {
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

// Returns all of the link paths for a plugin
func pluginPaths(pluginDir string, p *Plugin) []string {
	var paths []string

	paths = append(paths, util.ExecPluginPath(pluginDir, &p.GroupVersionKind))

	for _, v := range p.InstalledVersions {
		if v != p.Version {
			gvk := p.GroupVersionKind.DeepCopy()
			gvk.Version = v
			paths = append(paths, util.ExecPluginPath(pluginDir, gvk))
		}
	}

	return paths
}

// Checks to see if the plugin version is installed
func (p *Plugin) installed() bool {
	for _, v := range p.InstalledVersions {
		if v == p.Version {
			return true
		}
	}
	return false
}
