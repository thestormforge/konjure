/*
Copyright 2022 GramLabs, Inc.

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

package karg

import (
	"bytes"
	"os/exec"
	"slices"
	"strings"
)

type GetOption interface{ getCmd(*exec.Cmd) }
type CreateOption interface{ createCmd(*exec.Cmd) }
type ApplyOption interface{ applyCmd(*exec.Cmd) }
type DeleteOption interface{ deleteCmd(*exec.Cmd) }
type PatchOption interface{ patchCmd(*exec.Cmd) }
type WaitOption interface{ waitCmd(*exec.Cmd) }

func WithGetOptions(cmd *exec.Cmd, opts ...GetOption) {
	for _, opt := range opts {
		opt.getCmd(cmd)
	}
}

func WithCreateOptions(cmd *exec.Cmd, opts ...CreateOption) {
	for _, opt := range opts {
		opt.createCmd(cmd)
	}
}

func WithApplyOptions(cmd *exec.Cmd, opts ...ApplyOption) {
	for _, opt := range opts {
		opt.applyCmd(cmd)
	}
}

func WithDeleteOptions(cmd *exec.Cmd, opts ...DeleteOption) {
	for _, opt := range opts {
		opt.deleteCmd(cmd)
	}
}

func WithPatchOptions(cmd *exec.Cmd, opts ...PatchOption) {
	for _, opt := range opts {
		opt.patchCmd(cmd)
	}
}

func WithWaitOptions(cmd *exec.Cmd, opts ...WaitOption) {
	for _, opt := range opts {
		opt.waitCmd(cmd)
	}
}

// RawFile represents the "--filename=-" option along with the contents of stdin.
// NOTE: This option has no effect when used with an `ExecWriter` (which also uses `--filename=-`).
type RawFile []byte

func (o RawFile) applyCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o RawFile) kubectlCmd(cmd *exec.Cmd) {
	if len(o) > 0 {
		if !slices.Contains(cmd.Args, "--filename=-") {
			cmd.Args = append(cmd.Args, "--filename=-")
			cmd.Stdin = bytes.NewReader(o)
		}
	}
}

// For Resource and Selector there is more going on... i.e. this is what `kubectl get` says:
// (TYPE[.VERSION][.GROUP] [NAME | -l label] | TYPE[.VERSION][.GROUP]/NAME ...)

// Resource represents an individual named resource or type of resource passed as an argument.
type Resource string

func (o Resource) getCmd(cmd *exec.Cmd)   { o.kubectlCmd(cmd) }
func (o Resource) patchCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o Resource) kubectlCmd(cmd *exec.Cmd) {
	if o != "" {
		cmd.Args = append(cmd.Args, string(o))
	}
}

// ResourceType returns a resource argument using the GVR(s).
func ResourceType(resource ...string) Resource {
	return Resource(strings.Join(resource, ","))
}

// ResourceKind returns a resource argument using the GVK.
func ResourceKind(apiVersion, kind string) Resource {
	// This only works because kubectl will actually accept a kind instead of resource name
	group, version, ok := strings.Cut(apiVersion, "/")
	if !ok {
		group, version = version, group
	}
	return Resource(kind + "." + version + "." + group)
}

// ResourceName returns a resource argument using a GVR and a name.
func ResourceName(resourceType, resourceName string) Resource {
	if resourceName == "" {
		return Resource(resourceType)
	}
	return Resource(resourceType + "/" + resourceName)
}

// ResourceKindName returns a resource argument using GVK and a name.
func ResourceKindName(apiVersion, kind, name string) Resource {
	// This only works because kubectl will actually accept a kind instead of resource name
	group, version, ok := strings.Cut(apiVersion, "/")
	if !ok {
		group, version = version, group
	}
	return Resource(kind + "." + version + "." + group + "/" + name)
}

// Subresource represents the subresource passed as an argument.
type Subresource string

func (o Subresource) getCmd(cmd *exec.Cmd)   { o.kubectlCmd(cmd) }
func (o Subresource) applyCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o Subresource) patchCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o Subresource) kubectlCmd(cmd *exec.Cmd) {
	if o != "" {
		cmd.Args = append(cmd.Args, "--subresource", string(o))
	}
}

// AllNamespaces represents the "--all-namespaces" option.
type AllNamespaces bool

func (o AllNamespaces) getCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o AllNamespaces) kubectlCmd(cmd *exec.Cmd) {
	if o {
		// This is special in that we need to strip out an existing "--namespace" option
		args := make([]string, 0, len(cmd.Args)+1)
		for i := 0; i < len(cmd.Args); i++ {
			switch arg := cmd.Args[i]; arg {
			case "--namespace", "-n":
				i++
			default:
				if !strings.HasPrefix(arg, "--namespace=") && !strings.HasPrefix(arg, "-n=") {
					args = append(args, arg)
				}
			}
		}
		cmd.Args = append(args, "--all-namespaces")
	}
}

// Selector represents the "--selector" option.
type Selector string

func (o Selector) getCmd(cmd *exec.Cmd)    { o.kubectlCmd(cmd) }
func (o Selector) deleteCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o Selector) kubectlCmd(cmd *exec.Cmd) {
	if o != "" {
		cmd.Args = append(cmd.Args, "--selector", string(o))
	}
}

// DryRun represents the "--dry-run=none|client|server" option.
type DryRun string

func (o DryRun) createCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o DryRun) applyCmd(cmd *exec.Cmd)  { o.kubectlCmd(cmd) }
func (o DryRun) deleteCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o DryRun) patchCmd(cmd *exec.Cmd)  { o.kubectlCmd(cmd) }
func (o DryRun) kubectlCmd(cmd *exec.Cmd) {
	if o != "" {
		cmd.Args = append(cmd.Args, "--dry-run="+string(o))
	}
}

const (
	DryRunNone   DryRun = "none"
	DryRunClient DryRun = "client"
	DryRunServer DryRun = "server"
)

func (o DryRun) IsDryRun() bool { return o != "" && o != "none" }

// ServerSide represents the "--server-side" option.
type ServerSide bool

func (o ServerSide) applyCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o ServerSide) kubectlCmd(cmd *exec.Cmd) {
	if o {
		cmd.Args = append(cmd.Args, "--server-side")
	}
}

// ForceConflicts represents the "--force-conflicts" option.
type ForceConflicts bool

func (o ForceConflicts) applyCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o ForceConflicts) kubectlCmd(cmd *exec.Cmd) {
	if o {
		cmd.Args = append(cmd.Args, "--force-conflicts")
	}
}

// IgnoreNotFound represents the "--ignore-not-found" option.
type IgnoreNotFound bool

func (o IgnoreNotFound) getCmd(cmd *exec.Cmd)    { o.kubectlCmd(cmd) }
func (o IgnoreNotFound) deleteCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o IgnoreNotFound) kubectlCmd(cmd *exec.Cmd) {
	if o {
		cmd.Args = append(cmd.Args, "--ignore-not-found")
	}
}

// Output represents the "--output=json|yaml|wide|name|custom-columns=|custom-columns-file=|go-template=|go-template-file=|jsonpath=|jsonpath-file=" option.
type Output string

func (o Output) applyCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o Output) patchCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o Output) kubectlCmd(cmd *exec.Cmd) {
	if o != "" {
		cmd.Args = append(cmd.Args, "--output="+string(o))
	}
}

const (
	OutputJSON Output = "json"
	OutputYAML Output = "yaml"
	OutputWide Output = "wide"
	OutputName Output = "name"
)

func OutputCustomColumns(cols ...string) Output { // cols is <NAME>:<JSONPATH>
	return Output("custom-columns=" + strings.Join(cols, ","))
}
func OutputCustomColumnsFile(file string) Output { return Output("custom-columns-file=" + file) }
func OutputGoTemplate(tmpl string) Output        { return Output("go-template=" + tmpl) }
func OutputGoTemplateFile(file string) Output    { return Output("go-template-file=" + file) }
func OutputJSONPath(path string) Output          { return Output("jsonpath=" + path) }
func OutputJSONPathFile(file string) Output      { return Output("jsonpath-file=" + file) }

// Wait represents the "--wait" option
type Wait bool

func (o Wait) applyCmd(cmd *exec.Cmd)  { o.kubectlCmd(cmd) }
func (o Wait) deleteCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o Wait) kubectlCmd(cmd *exec.Cmd) {
	if o {
		cmd.Args = append(cmd.Args, "--wait")
	}
}

// FieldManager represents the "--field-manager" option.
type FieldManager string

func (o FieldManager) applyCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o FieldManager) patchCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o FieldManager) kubectlCmd(cmd *exec.Cmd) {
	if o != "" {
		cmd.Args = append(cmd.Args, "--field-manager", string(o))
	}
}

const (
	FieldManagerKubectlPatch FieldManager = "kubectl-patch"
)

// PatchType represents the "--type=json|merge|strategic" option on the patch command.
// NOTE: As a special case, an "apply" patch will produce a "--server-side" option instead.
type PatchType string

func (o PatchType) patchCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o PatchType) kubectlCmd(cmd *exec.Cmd) {
	if o != "" {
		// Allow the media type to be used interchangeably with the type arg values
		switch string(o) {
		case "application/json-patch+json":
			cmd.Args = append(cmd.Args, "--type=json")
		case "application/merge-patch+json":
			cmd.Args = append(cmd.Args, "--type=merge")
		case "application/strategic-merge-patch+json":
			cmd.Args = append(cmd.Args, "--type=strategic")
		default:
			cmd.Args = append(cmd.Args, "--type="+string(o))
		}
	}
}

const (
	PatchTypeJSON      PatchType = "json"
	PatchTypeMerge     PatchType = "merge"
	PatchTypeStrategic PatchType = "strategic"
)

// Patch represents the "--patch" option on the patch command.
type Patch string

func (o Patch) patchCmd(cmd *exec.Cmd) { o.kubectlCmd(cmd) }
func (o Patch) kubectlCmd(cmd *exec.Cmd) {
	if o != "" {
		// Windows does not support "extra files" so we probably can't use --patch-file to clean this up
		cmd.Args = append(cmd.Args, "--patch", string(o))
	}
}

// TODO Should we have something like a `func MarshalPatch(any) (Patch, error)` that marshals the JSON?
