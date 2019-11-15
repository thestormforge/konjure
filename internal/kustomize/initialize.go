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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/api/konfig"
)

// NewInitializeCommand returns a command for initializing Konjure.
func NewInitializeCommand() *cobra.Command {
	opts := &initializeOptions{}

	cmd := &cobra.Command{
		Use:          "init [PLUGIN...]",
		Short:        "Configure Kustomize plugins",
		Long:         "Manages your '~/.config/kustomize/plugin' directory to include symlinks back to the Konjure executable",
		SilenceUsage: true,
		PreRunE:      opts.preRun,
		RunE:         opts.run,
	}

	cmd.Flags().StringVar(&opts.PluginDir, "plugins", "", "override the `path` to the plugin directory")
	cmd.Flags().StringVar(&opts.Source, "source", "", "override the `path` to the source executable")
	cmd.Flags().BoolVar(&opts.Prune, "prune", false, "remove old versions")
	cmd.Flags().BoolVarP(&opts.Verbose, "verbose", "v", false, "be more verbose")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "check existing plugins")

	return cmd
}

type plugin struct {
	metav1.GroupVersionKind
	Supported bool
	Installed bool
}

type pluginStatus struct {
	Path        string
	PathRemoved bool
	PathCreated bool
	Source      string
}

type initializeOptions struct {
	PluginDir string
	Source    string
	Kinds     []string
	Prune     bool
	Verbose   bool
	DryRun    bool
}

func (o *initializeOptions) preRun(cmd *cobra.Command, args []string) error {
	// Capture the arguments as the kinds
	o.Kinds = args

	// Determine the directory where plugins are located
	if o.PluginDir == "" {
		pc, err := konfig.EnabledPluginConfig()
		if err != nil {
			return err
		}
		o.PluginDir = pc.AbsPluginHome
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

func (o *initializeOptions) run(cmd *cobra.Command, args []string) error {
	var commands []*cobra.Command
	if cmd.Parent() != nil {
		commands = cmd.Parent().Commands()
	}

	// Load and filter the plugin list
	plugins := loadPlugins(commands, o.PluginDir, !o.Prune)
	if len(o.Kinds) > 0 {
		plugins = filterPlugins(plugins, o.Kinds)
	}

	tw := tabwriter.NewWriter(cmd.OutOrStdout(), 1, 2, 2, '.', 0)
	defer func() { _ = tw.Flush() }()

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

func (o *initializeOptions) createLinks(p *plugin) (*pluginStatus, error) {
	status := &pluginStatus{Path: pluginPath(o.PluginDir, &p.GroupVersionKind)}
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

func filterPlugins(plugins []plugin, filters []string) []plugin {
	var filtered []plugin
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

func loadPlugins(commands []*cobra.Command, pluginDir string, keepAllVersions bool) []plugin {
	var plugins []plugin
	groups := make(map[string]bool)

	// First load the currently supported plugins
	for _, c := range commands {
		gvk := commandGVK(c)
		if gvk != nil {
			groups[gvk.Group] = true
			plugins = append(plugins, plugin{GroupVersionKind: *gvk, Supported: true})
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
				plugins = append(plugins, plugin{GroupVersionKind: *gvk})
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

// pluginPath returns the path to an executable plugin
func pluginPath(pluginDir string, gvk *metav1.GroupVersionKind) string {
	return filepath.Join(pluginDir, gvk.Group, gvk.Version, strings.ToLower(gvk.Kind), gvk.Kind)
}

// commandGVK returns the GVK for the supplied executable plugin command; returns nil if the command is not an executable plugin
func commandGVK(cmd *cobra.Command) *metav1.GroupVersionKind {
	if cmd.Annotations["group"] == "" || cmd.Annotations["version"] == "" || cmd.Annotations["kind"] == "" {
		return nil
	}
	return &metav1.GroupVersionKind{
		Group:   cmd.Annotations["group"],
		Version: cmd.Annotations["version"],
		Kind:    cmd.Annotations["kind"],
	}
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
