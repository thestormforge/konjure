package internal

import (
	_ "embed"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"
)

//go:embed application_schema.json
var applicationSchema []byte

func init() {
	// Add our own schema that includes the merge directives necessary for getting
	// correct behavior out of the merges.
	if err := openapi.AddSchema(applicationSchema); err != nil {
		panic(err)
	}
}

// IndexApplications moves the application resources from a collection of nodes
// and into an indexed map.
func IndexApplications(nodes []*yaml.RNode, apps map[yaml.NameMeta]*ApplicationNode) ([]*yaml.RNode, error) {
	var i int
	for _, node := range nodes {
		md, err := node.GetMeta()
		if err != nil {
			return nil, err
		}

		// Leave non-Application nodes alone
		if md.APIVersion != "app.k8s.io/v1beta1" || md.Kind != "Application" {
			nodes[i] = node
			i++
			continue
		}

		// Find or insert the application in the map
		app := apps[md.NameMeta]
		if app == nil {
			app = &ApplicationNode{namespace: md.Namespace}
			apps[md.NameMeta] = app
		}

		// Update the application with the new node information
		app.Node, err = node.Pipe(
			yaml.FilterFunc(func(object *yaml.RNode) (*yaml.RNode, error) {
				return merge2.Merge(app.Node, object, yaml.MergeOptions{})
			}),

			yaml.Tee(
				yaml.Lookup("spec", "selector"),
				yaml.FilterFunc(func(object *yaml.RNode) (*yaml.RNode, error) {
					s := &LabelSelector{}
					if err := object.YNode().Decode(s); err != nil {
						return nil, err
					}
					app.selector = s.String()
					return nil, nil
				}),
			),

			yaml.Tee(
				yaml.Lookup("spec", "componentKinds"),
				yaml.FilterFunc(func(object *yaml.RNode) (*yaml.RNode, error) {
					return nil, object.YNode().Decode(&app.componentKinds)
				}),
			),
		)
		if err != nil {
			return nil, err
		}
	}
	return nodes[:i], nil
}

type ApplicationNode struct {
	Node           *yaml.RNode
	namespace      string
	componentKinds []GroupKind
	selector       string
}

// Filter removes all the application resources from the supplied collection.
func (app *ApplicationNode) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	result := make([]*yaml.RNode, 0, len(nodes))
	for _, node := range nodes {
		md, err := node.GetMeta()
		if err != nil {
			return nil, err
		}

		owns, err := app.owns(md, node)
		if err != nil {
			return nil, err
		}

		if !owns {
			result = append(result, node)
		}
	}
	return result, nil
}

func (app *ApplicationNode) owns(md yaml.ResourceMeta, node *yaml.RNode) (bool, error) {
	// The node must be in the same namespace as the application
	if md.Namespace != app.namespace {
		return false, nil
	}

	// The node type must be in the list of application component kinds
	var kindMatch bool
	for i := range app.componentKinds {
		if app.componentKinds[i].Matches(md.TypeMeta) {
			kindMatch = true
			break
		}
	}
	if !kindMatch {
		return false, nil
	}

	// The node must match the application label selector
	if ok, err := node.MatchesLabelSelector(app.selector); err != nil || !ok {
		return false, err
	}

	return true, nil
}

type GroupKind struct {
	Group string `yaml:"group"`
	Kind  string `yaml:"kind"`
}

// Note that the Application SIG shows this case as both "v1" and "core",
// however something like `kubectl get Service.core` or `kubectl get Service.v1`
// will fail while `kubectl get Service.` works

func GroupKindFromType(t yaml.TypeMeta) GroupKind {
	g := path.Dir(t.APIVersion)
	if g == "." {
		g = ""
	}

	return GroupKind{
		Group: g,
		Kind:  t.Kind,
	}
}

func (gk *GroupKind) Matches(t yaml.TypeMeta) bool {
	if t.Kind != gk.Kind {
		return false
	}

	g := path.Dir(t.APIVersion)
	if g == "." {
		g = ""
	}

	return g == gk.Group
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

// SplitHelmChart splits a Helm chart into it's name and version.
func SplitHelmChart(chart string) (name string, version string) {
	name = regexp.MustCompile(`-(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)`).Split(chart, 2)[0]
	version = strings.TrimPrefix(strings.TrimPrefix(chart, name), "-")
	return
}
