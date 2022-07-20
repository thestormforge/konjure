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

package v1beta2

import (
	"github.com/sethvargo/go-password/password"
)

// Resource is used to expand a list of URL-like specifications into other Konjure resources.
type Resource struct {
	// The list of URL-like specifications to convert into Konjure resources.
	Resources []string `json:"resources" yaml:"resources"`
}

// HelmValue specifies a value or value file for configuring a Helm chart.
type HelmValue struct {
	// Path to a values.yaml file.
	File string `json:"file,omitempty" yaml:"file,omitempty"`
	// Name of an individual name/value to set.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// Value of an individual name/value to set.
	Value string `json:"value,omitempty" yaml:"value,omitempty"`
	// Flag indicating that numeric like value should be quoted as strings (e.g. for environment variables).
	ForceString bool `json:"forceString,omitempty" yaml:"forceString,omitempty"` // TODO Eliminate and use IntOrString?
	// Treat value as a file and load the contents in place of the actual value.
	LoadFile bool `json:"loadFile,omitempty" yaml:"loadFile,omitempty"`
}

// Helm is used to expand a Helm chart locally (using `helm template`).
type Helm struct {
	// The release name to use when rendering the chart templates.
	ReleaseName string `json:"releaseName,omitempty" yaml:"releaseName,omitempty"`
	// The namespace to use when rendering the chart templates (this is particularly important for charts that
	// may produce resources in multiple namespaces).
	ReleaseNamespace string `json:"releaseNamespace,omitempty" yaml:"releaseNamespace,omitempty"`
	// The chart name to inflate.
	Chart string `json:"chart" yaml:"chart"`
	// The specific version of the chart to use (defaults to the latest release).
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// The repository URL to get the chart from.
	Repository string `json:"repo" yaml:"repo"`
	// The values used to configure the chart.
	Values []HelmValue `json:"values,omitempty" yaml:"values,omitempty"`
	// Flag to filter out tests from the results.
	IncludeTests bool `json:"includeTests,omitempty" yaml:"includeTests,omitempty"`
}

// JsonnetParameter specifies inputs to a Jsonnet program.
type JsonnetParameter struct {
	// The name of the parameter.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// The string value of the parameter.
	String string `json:"string,omitempty" yaml:"string,omitempty"`
	// The file name containing a string parameter value.
	StringFile string `json:"stringFile,omitempty" yaml:"stringFile,omitempty"`
	// Code to include.
	Code string `json:"code,omitempty" yaml:"code,omitempty"`
	// The file name containing code to include.
	CodeFile string `json:"codeFile,omitempty" yaml:"codeFile,omitempty"`
}

// Jsonnet is used to expand programmatically constructed resources.
type Jsonnet struct {
	// The Jsonnet file to evaluate.
	Filename string `json:"filename,omitempty" yaml:"filename,omitempty"`
	// An anonymous code snippet to evaluate.
	Code string `json:"exec,omitempty" yaml:"exec,omitempty"`
	// Additional directories to consider when importing additional Jsonnet code.
	JsonnetPath []string `json:"jpath,omitempty" yaml:"jpath,omitempty"`
	// The list of external variables to evaluate against.
	ExternalVariables []JsonnetParameter `json:"extVar,omitempty" yaml:"extVar,omitempty"`
	// The list of top level arguments to evaluate against.
	TopLevelArguments []JsonnetParameter `json:"topLevelArg,omitempty" yaml:"topLevelArg,omitempty"`

	// Explicit directory to use fo Jsonnet Bundler support (defaults to "vendor" if "jsonnetfile.json" is present).
	JsonnetBundlerPackageHome string `json:"jbPkgHome,omitempty" yaml:"jbPkgHome,omitempty"`
	// Flag to force a Bundler refresh, even if the package home directory is already present.
	JsonnetBundlerRefresh bool `json:"jbRefresh,omitempty" yaml:"jbRefresh,omitempty"`
}

// Kubernetes is used to expand resources found in a Kubernetes cluster.
type Kubernetes struct {
	// The namespace to look for resources in.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	// An explicit list of namespaces to look for resources in.
	Namespaces []string `json:"namespaces,omitempty" yaml:"namespaces,omitempty"`
	// A label selector matching namespaces to look for resources in.
	NamespaceSelector string `json:"namespaceSelector,omitempty" yaml:"namespaceSelector,omitempty"`
	// The list of resource types to include. Defaults to "deployments,statefulsets,configmaps".
	Types []string `json:"types,omitempty" yaml:"types,omitempty"`
	// A label selector to limit which resources are included. Defaults to "" (match everything).
	Selector string `json:"selector,omitempty" yaml:"selector,omitempty"`
	// A field selector to limit which resources are included. Defaults to "" (match everything).
	FieldSelector string `json:"fieldSelector,omitempty" yaml:"fieldSelector,omitempty"`
}

// Kustomize is used to expand kustomizations.
type Kustomize struct {
	// The Kustomize root to build.
	Root string `json:"root" yaml:"root"`
}

// PasswordRecipe is used to configure random password strings for secrets.
type PasswordRecipe struct {
	// The key in the secret data field to use.
	Key string `json:"key" yaml:"key"`
	// The length of the password.
	Length *int `json:"length,omitempty" yaml:"length,omitempty"`
	// The number of digits to include in the password.
	NumDigits *int `json:"numDigits,omitempty" yaml:"numDigits,omitempty"`
	// The number of symbol characters to include in the password.
	NumSymbols *int `json:"numSymbols,omitempty" yaml:"numSymbols,omitempty"`
	// Flag restricting the use of uppercase characters.
	NoUpper *bool `json:"noUpper,omitempty" yaml:"noUpper,omitempty"`
	// Flag restricting repeating characters.
	AllowRepeat *bool `json:"allowRepeat,omitempty" yaml:"allowRepeat,omitempty"`
}

// Secret is used to expand a Secret resource.
type Secret struct {
	// The name of the secret to generate.
	SecretName string `json:"secretName" yaml:"secretName"`
	// The type of secret to generate.
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// A list of `key=value` pairs to include on the secret.
	LiteralSources []string `json:"literals,omitempty" yaml:"literals,omitempty"`
	// A list of files (or `key=filename` pairs) to include on the secret.
	FileSources []string `json:"files,omitempty" yaml:"files,omitempty"`
	// A list of .env files (files containing `key=value` pairs) to include on the secret.
	EnvSources []string `json:"envs,omitempty" yaml:"envs,omitempty"`
	// A list of keys to include randomly generated UUIDs for on the secret.
	UUIDSources []string `json:"uuids,omitempty" yaml:"uuids,omitempty"`
	// A list of keys to include randomly generated ULIDs for on the secret.
	ULIDSources []string `json:"ulids,omitempty" yaml:"ulids,omitempty"`
	// A list of password recipes to include random strings on the secret.
	PasswordSources []PasswordRecipe `json:"passwords,omitempty" yaml:"passwords,omitempty"`

	// Additional configuration for generating passwords.
	PasswordOptions *password.GeneratorInput `json:"-" yaml:"-"`
}

// Git is used to expand full or partial Git repositories.
type Git struct {
	// The Git repository URL.
	Repository string `json:"repo,omitempty" yaml:"repo,omitempty"`
	// The refspec in the repository to checkout.
	Refspec string `json:"refspec,omitempty" yaml:"refspec,omitempty"`
	// The subdirectory context to limit the Git repository to.
	Context string `json:"context,omitempty" yaml:"context,omitempty"`
}

// HTTP is used to expand HTTP resources.
type HTTP struct {
	// The HTTP(S) URL to fetch.
	URL string `json:"url" yaml:"url"`
}

// File is used to expand local file system resources.
type File struct {
	// The file (or directory) name to read.
	Path string `json:"path" yaml:"path"`
}
