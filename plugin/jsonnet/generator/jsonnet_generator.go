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

package generator

import (
	"github.com/carbonrelay/konjure/internal/jsonnet"
	"sigs.k8s.io/kustomize/v3/pkg/ifc"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/yaml"
)

type plugin struct {
	ldr ifc.Loader
	rf  *resmap.Factory

	Jsonnet           jsonnet.Executor    `json:"jsonnet"`
	Filename          string              `json:"filename"`
	Code              string              `json:"exec"`
	JsonnetPath       []string            `json:"jpath"`
	ExternalVariables []jsonnet.Parameter `json:"extVar"`
	TopLevelArguments []jsonnet.Parameter `json:"topLevelArg"`
}

var KustomizePlugin plugin

func (p *plugin) Config(ldr ifc.Loader, rf *resmap.Factory, c []byte) error {
	p.ldr = ldr
	p.rf = rf
	return yaml.Unmarshal(c, p)
}

func (p *plugin) Generate() (resmap.ResMap, error) {
	p.Jsonnet.Complete()

	b, err := p.execute()
	if err != nil {
		return nil, err
	}

	m := resmap.New()
	if err := jsonnet.AppendMultiDocumentJSONBytes(p.rf.RF(), m, b); err != nil {
		return nil, err
	}

	return m, nil
}

func (p *plugin) execute() ([]byte, error) {
	if p.Filename != "" {
		return p.Jsonnet.ExecuteFile(p.Filename, p.JsonnetPath, p.ExternalVariables, p.TopLevelArguments)
	}

	if p.Code != "" {
		return p.Jsonnet.ExecuteCode(p.Code, p.JsonnetPath, p.ExternalVariables, p.TopLevelArguments)
	}

	return nil, nil
}
