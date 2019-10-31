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
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/carbonrelay/konjure/plugin/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kplug "sigs.k8s.io/kustomize/v3/pkg/plugins"
)

type Plugin struct {
	metav1.GroupVersionKind
	Supported bool
	Installed bool
}

type PluginStatus struct {
	Path        string
	PathRemoved bool
	PathCreated bool
	Source      string
}

type InitializeOptions struct {
	PluginDir string
	Source    string
	Kinds     []string
	Prune     bool
	Verbose   bool
	DryRun    bool
}

func NewInitializeOptions() *InitializeOptions {
	return &InitializeOptions{}
}

func (o *InitializeOptions) Complete() error {
	// Determine the directory where plugins are located
	if o.PluginDir == "" {
		o.PluginDir = kplug.ActivePluginConfig().DirectoryPath
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
	plugins := LoadPlugins(o.PluginDir, !o.Prune)
	if len(o.Kinds) > 0 {
		plugins = FilterPlugins(plugins, o.Kinds)
	}

	tw := tabwriter.NewWriter(out, 1, 2, 2, '.', 0)
	defer tw.Flush()

	// Process each plugin
	for i := range plugins {
		status, err := o.createLinks(&plugins[i])
		if err != nil {
			return err
		}

		summary := "OK"
		if status.PathCreated && status.PathRemoved {
			summary = "Updated"
		} else if status.PathCreated {
			summary = "Created"
		} else if status.PathRemoved {
			summary = "Removed"
		}

		if o.Verbose {
			_, _ = fmt.Fprintf(tw, "%s\t%s\n", status.Path, summary)
		} else {
			_, _ = fmt.Fprintf(tw, "%s/%s\t%s\n", plugins[i].Kind, plugins[i].Version, summary)
		}
	}

	return nil
}

func (o *InitializeOptions) createLinks(p *Plugin) (*PluginStatus, error) {
	status := &PluginStatus{Path: util.ExecPluginPath(o.PluginDir, &p.GroupVersionKind)}
	dir := filepath.Dir(status.Path)

	// Remove unsupported plugins
	if !p.Supported {
		status.PathRemoved = true
		if o.DryRun {
			return status, nil
		}
		return status, os.RemoveAll(dir)
	}

	// Create missing directories unless this is a dry run
	if !o.DryRun {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
	}

	// If possible, read the existing link
	if l, err := os.Readlink(status.Path); err == nil {
		status.Source = l
	}

	// If it already matches, there is nothing to do
	if status.Source == o.Source {
		return status, nil
	}

	// Do not actually create the links if this a dry run
	status.PathCreated = true
	if o.DryRun {
		if _, err := os.Lstat(status.Path); err == nil {
			status.PathRemoved = true
		}
		return status, nil
	}

	// Remove any existing file and link it
	status.PathRemoved = true
	if err := os.Remove(status.Path); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		status.PathRemoved = false
	}
	return status, os.Symlink(o.Source, status.Path)
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

func LoadPlugins(pluginDir string, keepAllVersions bool) []Plugin {
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
	for g := range groups {
		_ = filepath.Walk(filepath.Join(pluginDir, g), func(path string, _ os.FileInfo, _ error) error {
			gvk := pluginGVK(pluginDir, path)
			if gvk == nil {
				return nil
			}

			var installed bool
			for i := range plugins {
				if plugins[i].Group == gvk.Group && plugins[i].Kind == gvk.Kind {
					plugins[i].Supported = plugins[i].Supported || keepAllVersions
				}
				if plugins[i].Group == gvk.Group && plugins[i].Version == gvk.Version && plugins[i].Kind == gvk.Kind {
					plugins[i].Installed = true
					installed = true
				}
			}

			if !installed {
				plugins = append(plugins, Plugin{GroupVersionKind: *gvk})
			}
			return nil
		})
	}

	// Finally, sort by kind
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Kind < plugins[j].Kind
	})

	return plugins
}

// Returns the GVK for a path given a base directory if possible
func pluginGVK(pluginDir, path string) *metav1.GroupVersionKind {
	rel, err := filepath.Rel(pluginDir, path)
	if err != nil {
		return nil
	}

	gvk := metav1.GroupVersionKind{}
	rel, gvk.Kind = filepath.Split(filepath.Clean(rel))
	rel, lowerKind := filepath.Split(filepath.Clean(rel))
	rel, gvk.Version = filepath.Split(filepath.Clean(rel))
	gvk.Group = filepath.Clean(rel)

	if lowerKind != strings.ToLower(gvk.Kind) {
		return nil
	}
	return &gvk
}
