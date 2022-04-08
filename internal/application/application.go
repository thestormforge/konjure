package application

import (
	_ "embed"

	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"
)

//go:embed application_schema.json
var schema []byte

func init() {
	// Add our own schema that includes the merge directives necessary for getting
	// correct behavior out of the merges.
	if err := openapi.AddSchema(schema); err != nil {
		panic(err)
	}
}

// Index moves the application resources from a collection of nodes
// and into an indexed map.
func Index(nodes []*yaml.RNode, apps map[yaml.NameMeta]*Node) ([]*yaml.RNode, error) {
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
			app = &Node{namespace: md.Namespace}
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

type Node struct {
	Node           *yaml.RNode
	namespace      string
	componentKinds []GroupKind
	selector       string
}

// Filter removes all the application resources from the supplied collection.
func (app *Node) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
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

func (app *Node) owns(md yaml.ResourceMeta, node *yaml.RNode) (bool, error) {
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
