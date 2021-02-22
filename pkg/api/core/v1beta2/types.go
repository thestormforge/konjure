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
	"net/url"

	"github.com/sethvargo/go-password/password"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Resource struct {
	yaml.ResourceMeta `json:",inline" yaml:",inline"`

	Resources []string `json:"resources" yaml:"resources"`
}

type HelmExecutor struct {
	Bin             string `json:"bin,omitempty" yaml:"bin,omitempty"`
	RepositoryCache string `json:"repositoryCache,omitempty" yaml:"repositoryCache,omitempty"`
}

type HelmValue struct {
	File        string `json:"file,omitempty" yaml:"file,omitempty"`
	Name        string `json:"name,omitempty" yaml:"name,omitempty"`
	Value       string `json:"value,omitempty" yaml:"value,omitempty"`
	ForceString bool   `json:"forceString,omitempty" yaml:"forceString,omitempty"` // TODO Eliminate and use IntOrString?
	LoadFile    bool   `json:"loadFile,omitempty" yaml:"loadFile,omitempty"`
}

type Helm struct {
	yaml.ResourceMeta `json:",inline" yaml:",inline"`

	Helm             HelmExecutor `json:"helm,omitempty" yaml:"helm,omitempty"`
	ReleaseName      string       `json:"releaseName,omitempty" yaml:"releaseName,omitempty"`
	ReleaseNamespace string       `json:"releaseNamespace,omitempty" yaml:"releaseNamespace,omitempty"`
	Chart            string       `json:"chart" yaml:"chart"`
	Version          string       `json:"version,omitempty" yaml:"version,omitempty"`
	Repository       string       `json:"repo" yaml:"repo"`
	Values           []HelmValue  `json:"values,omitempty" yaml:"values,omitempty"`
	IncludeTests     bool         `json:"includeTests,omitempty" yaml:"includeTests,omitempty"`
}

type JsonnetParameter struct {
	Name       string `json:"name,omitempty" yaml:"name,omitempty"`
	String     string `json:"string,omitempty" yaml:"string,omitempty"`
	StringFile string `json:"stringFile,omitempty" yaml:"stringFile,omitempty"`
	Code       string `json:"code,omitempty" yaml:"code,omitempty"`
	CodeFile   string `json:"codeFile,omitempty" yaml:"codeFile,omitempty"`
}

type Jsonnet struct {
	yaml.ResourceMeta `json:",inline" yaml:",inline"`

	Filename          string             `json:"filename" yaml:"filename"`
	Code              string             `json:"exec" yaml:"exec"`
	JsonnetPath       []string           `json:"jpath" yaml:"jpath"`
	ExternalVariables []JsonnetParameter `json:"extVar" yaml:"extVar"`
	TopLevelArguments []JsonnetParameter `json:"topLevelArg" yaml:"topLevelArg"`

	JsonnetBundlerPackageHome string `json:"jbPkgHome" yaml:"jbPkgHome"`
	JsonnetBundlerRefresh     bool   `json:"jbRefresh" yaml:"jbRefresh"`
}

// TODO Should we also have a namespace list?
type KubernetesSelector struct {
	Types             []string `json:"types" yaml:"types"`
	NamespaceSelector string   `json:"namespaceSelector,omitempty" yaml:"namespaceSelector,omitempty"`
	LabelSelector     string   `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty"`
}

type Kubernetes struct {
	yaml.ResourceMeta `json:",inline" yaml:",inline"`

	Bin        string `json:"bin,omitempty" yaml:"bin,omitempty"`
	Kubeconfig string `json:"kubeconfig,omitempty" yaml:"kubeconfig,omitempty"`
	Context    string `json:"context,omitempty" yaml:"context,omitempty"`

	Resources []KubernetesSelector `json:"resources" yaml:"resources"`
}

type Kustomize struct {
	yaml.ResourceMeta `json:",inline" yaml:",inline"`

	Root string `json:"root" yaml:"root"`
}

type PasswordRecipe struct {
	Key         string `json:"key" yaml:"key"`
	Length      *int   `json:"length,omitempty" yaml:"length,omitempty"`
	NumDigits   *int   `json:"numDigits,omitempty" yaml:"numDigits,omitempty"`
	NumSymbols  *int   `json:"numSymbols,omitempty" yaml:"numSymbols,omitempty"`
	NoUpper     *bool  `json:"noUpper,omitempty" yaml:"noUpper,omitempty"`
	AllowRepeat *bool  `json:"allowRepeat,omitempty" yaml:"allowRepeat,omitempty"`
}

type Secret struct {
	yaml.ResourceMeta `json:",inline" yaml:",inline"`

	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	LiteralSources  []string         `json:"literals,omitempty" yaml:"literals,omitempty"`
	FileSources     []string         `json:"files,omitempty" yaml:"files,omitempty"`
	EnvSources      []string         `json:"envs,omitempty" yaml:"envs,omitempty"`
	UUIDSources     []string         `json:"uuids,omitempty" yaml:"uuids,omitempty"`
	ULIDSources     []string         `json:"ulids,omitempty" yaml:"ulids,omitempty"`
	PasswordSources []PasswordRecipe `json:"passwords,omitempty" yaml:"passwords,omitempty"`

	PasswordOptions *password.GeneratorInput `json:"passwordOptions,omitempty" yaml:"passwordOptions,omitempty"`
}

type Git struct {
	yaml.ResourceMeta `json:",inline" yaml:",inline"`

	Repository url.URL
	Refspec    string
	Context    string
}

type HTTP struct {
	yaml.ResourceMeta `json:",inline" yaml:",inline"`

	URL string `json:"url" yaml:"url"`
}

type File struct {
	yaml.ResourceMeta `json:",inline" yaml:",inline"`

	Name string `json:"name" yaml:"name"`
}
