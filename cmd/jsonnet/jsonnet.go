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

package jsonnet

import (
	"io"

	"sigs.k8s.io/kustomize/v3/pkg/resmap"
)

// JsonnetOptions is the configuration for executing Jsonnet code
type JsonnetOptions struct {
	Jsonnet           Jsonnet     `json:"jsonnet"`
	Filename          string      `json:"filename"`
	Code              string      `json:"exec"`
	JsonnetPath       []string    `json:"jpath"`
	ExternalVariables []Parameter `json:"extVar"`
	TopLevelArguments []Parameter `json:"topLevelArg"`
}

func NewJsonnetOptions() *JsonnetOptions {
	return &JsonnetOptions{}
}

func (o *JsonnetOptions) Run(stderr io.Writer) ([]byte, error) {
	var m resmap.ResMap
	var err error
	if o.Filename != "" {
		m, err = o.Jsonnet.ExecuteFile(o.Filename, o.JsonnetPath, o.ExternalVariables, o.TopLevelArguments, stderr)
	} else if o.Code != "" {
		m, err = o.Jsonnet.ExecuteCode(o.Code, o.JsonnetPath, o.ExternalVariables, o.TopLevelArguments, stderr)
	} else {
		m = resmap.New()
	}
	if err != nil {
		return nil, err
	}

	return m.AsYaml()
}
