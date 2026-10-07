/*
Copyright 2022 CloudBolt, Inc.

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

package application

import (
	"fmt"
	"sort"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// StripVersion removes the version from a group/version (APIVersion) string.
func StripVersion(gv string) string {
	if gv == "" || gv == "v1" {
		return ""
	}
	return strings.Split(gv, "/")[0]
}

type GroupKind struct {
	Group string `yaml:"group"`
	Kind  string `yaml:"kind"`
}

// Note that the Application SIG shows the core type case as both "v1" and "core",
// however something like `kubectl get Service.core` or `kubectl get Service.v1`
// will fail while `kubectl get Service.` works. Therefore, empty string.

func (gk *GroupKind) Matches(t yaml.TypeMeta) bool {
	if t.Kind != gk.Kind {
		return false
	}

	if StripVersion(t.APIVersion) != gk.Group {
		return false
	}

	return true
}

func (gk *GroupKind) String() string {
	return gk.Kind + "." + gk.Group
}

type LabelSelector struct {
	MatchLabels      map[string]string `yaml:"matchLabels"`
	MatchExpressions []struct {
		Key      string   `yaml:"key"`
		Operator string   `yaml:"operator"`
		Values   []string `yaml:"values"`
	} `yaml:"matchExpressions"`
}

func (ls *LabelSelector) String() string {
	if ls == nil || len(ls.MatchLabels)+len(ls.MatchExpressions) == 0 {
		return ""
	}
	var req []string
	for k, v := range ls.MatchLabels {
		req = append(req, fmt.Sprintf("%s=%s", k, v))
	}
	for _, expr := range ls.MatchExpressions {
		switch expr.Operator {
		case "In":
			req = append(req, fmt.Sprintf("%s in (%s)", expr.Key, joinSorted(expr.Values, ",")))
		case "NotIn":
			req = append(req, fmt.Sprintf("%s notin (%s)", expr.Key, joinSorted(expr.Values, ",")))
		case "Exists":
			req = append(req, fmt.Sprintf("%s", expr.Key))
		case "DoesNotExist":
			req = append(req, fmt.Sprintf("!%s", expr.Key))
		}
	}
	return strings.Join(req, ",")
}

func joinSorted(values []string, sep string) string {
	if !sort.StringsAreSorted(values) {
		sorted := make([]string, len(values))
		copy(sorted, values)
		sort.Strings(sorted)
		values = sorted
	}
	return strings.Join(values, sep)
}
