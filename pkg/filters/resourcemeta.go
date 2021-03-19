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

package filters

import (
	"regexp"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ResourceMetaFilter filters nodes based on their resource metadata using regular
// expressions or Kubernetes selectors.
type ResourceMetaFilter struct {
	// Regular expression matching the group.
	Group string `json:"group,omitempty" yaml:"group,omitempty"`
	// Regular expression matching the version.
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// Regular expression matching the kind.
	Kind string `json:"kind,omitempty" yaml:"kind,omitempty"`
	// Regular expression matching the namespace.
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	// Regular expression matching the name.
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// Kubernetes selector matching labels.
	LabelSelector string `json:"labelSelector,omitempty" yaml:"labelSelector,omitempty"`
	// Kubernetes selector matching annotations
	AnnotationSelector string `json:"annotationSelector,omitempty" yaml:"annotationSelector,omitempty"`
}

func (f *ResourceMetaFilter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	m, err := newMetaMatcher(f)
	if err != nil {
		return nil, err
	}

	if m == nil && f.LabelSelector == "" && f.AnnotationSelector == "" {
		return nodes, nil
	}

	result := make([]*yaml.RNode, 0, len(nodes))
	for _, n := range nodes {
		if m != nil {
			if meta, err := n.GetMeta(); err != nil {
				return nil, err
			} else if !m.matchesMeta(meta) {
				continue
			}
		}

		if f.LabelSelector != "" {
			if matched, err := n.MatchesLabelSelector(f.LabelSelector); err != nil {
				return nil, err
			} else if !matched {
				continue
			}
		}

		if f.AnnotationSelector != "" {
			if matched, err := n.MatchesAnnotationSelector(f.AnnotationSelector); err != nil {
				return nil, err
			} else if !matched {
				continue
			}
		}

		result = append(result, n)
	}

	return result, nil
}

type metaMatcher struct {
	namespaceRegex *regexp.Regexp
	nameRegex      *regexp.Regexp
	groupRegex     *regexp.Regexp
	versionRegex   *regexp.Regexp
	kindRegex      *regexp.Regexp
}

func newMetaMatcher(g *ResourceMetaFilter) (m *metaMatcher, err error) {
	m = &metaMatcher{}
	notEmpty := false

	m.namespaceRegex, err = compileAnchored(g.Namespace)
	if err != nil {
		return nil, err
	}
	notEmpty = notEmpty || m.namespaceRegex != nil

	m.nameRegex, err = compileAnchored(g.Name)
	if err != nil {
		return nil, err
	}
	notEmpty = notEmpty || m.nameRegex != nil

	m.groupRegex, err = compileAnchored(g.Group)
	if err != nil {
		return nil, err
	}
	notEmpty = notEmpty || m.groupRegex != nil

	m.versionRegex, err = compileAnchored(g.Version)
	if err != nil {
		return nil, err
	}
	notEmpty = notEmpty || m.versionRegex != nil

	m.kindRegex, err = compileAnchored(g.Kind)
	if err != nil {
		return nil, err
	}
	notEmpty = notEmpty || m.kindRegex != nil

	if notEmpty {
		return m, nil
	}
	return nil, nil
}

func (m *metaMatcher) matchesMeta(meta yaml.ResourceMeta) bool {
	if m.namespaceRegex != nil && !m.namespaceRegex.MatchString(meta.Namespace) {
		return false
	}
	if m.nameRegex != nil && !m.nameRegex.MatchString(meta.Name) {
		return false
	}

	if m.groupRegex != nil || m.versionRegex != nil {
		group, version := "", meta.APIVersion
		if pos := strings.Index(version, "/"); pos >= 0 {
			group, version = version[0:pos], version[pos+1:]
		}

		if m.groupRegex != nil && !m.groupRegex.MatchString(group) {
			return false
		}

		if m.versionRegex != nil && !m.versionRegex.MatchString(version) {
			return false
		}
	}

	if m.kindRegex != nil && !m.kindRegex.MatchString(meta.Kind) {
		return false
	}

	return true
}

func compileAnchored(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}
	return regexp.Compile("^(?:" + pattern + ")$")
}
