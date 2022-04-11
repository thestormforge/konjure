package filters

import (
	"regexp"
	"strings"

	"github.com/thestormforge/konjure/internal/application"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ApplicationFilter produces applications based on the observed resources.
type ApplicationFilter struct {
	// Flag indicating if this filter should act as a pass-through.
	Enabled bool
	// Flag indicating we should show resources which do not belong to an application.
	ShowUnownedResources bool
	// The name of the label that contains the application name, default is "app.kubernetes.io/name".
	ApplicationNameLabel string
	// The name of the label that contains the application instance name, default is "app.kubernetes.io/instance".
	ApplicationInstanceLabel string
}

// Filter keeps all the application resources and creates application resources
// for all other nodes that are not associated with an application.
func (f *ApplicationFilter) Filter(nodes []*yaml.RNode) ([]*yaml.RNode, error) {
	if !f.Enabled {
		return nodes, nil
	}

	apps := make(map[yaml.NameMeta]*application.Node)
	var err error
	var scannedAppLabels bool

IndexApps:

	// Index the existing applications
	nodes, err = application.Index(nodes, apps)
	if err != nil {
		return nil, err
	}

	// Remove all the resources that belong to an existing application
	for _, app := range apps {
		nodes, err = app.Filter(nodes)
		if err != nil {
			return nil, err
		}
	}

	// Try to create applications from resource labels
	if !scannedAppLabels {
		var appsFromLabels []*yaml.RNode
		for _, node := range nodes {
			app, err := f.appFromLabels(node)
			if err != nil {
				return nil, err
			}
			if app != nil {
				appsFromLabels = append(appsFromLabels, app)
			}
		}

		scannedAppLabels = true
		if len(appsFromLabels) > 0 {
			nodes = append(nodes, appsFromLabels...)
			goto IndexApps
		}
	}

	// Drop the resources that weren't owned by an application
	if !f.ShowUnownedResources {
		nodes = nil
	}

	// Add the applications to list of remaining nodes
	for _, app := range apps {
		nodes = append(nodes, app.Node)
	}
	return nodes, nil
}

// appFromLabels attempts to use the recommended `app.kubernetes.io/*` labels
// from the supplied resource node to generate a new application node. If an
// application cannot be created, this function will just return nil.
func (f *ApplicationFilter) appFromLabels(n *yaml.RNode) (*yaml.RNode, error) {
	md, err := n.GetMeta()
	if err != nil {
		return nil, err
	}

	nameLabelKey := f.ApplicationNameLabel
	if nameLabelKey == "" {
		nameLabelKey = application.LabelName
	}

	instanceLabelKey := f.ApplicationInstanceLabel
	if instanceLabelKey == "" {
		instanceLabelKey = application.LabelInstance
	}

	nameLabel := md.Labels[nameLabelKey]
	instanceLabel := md.Labels[instanceLabelKey]
	partOfLabel := md.Labels[application.LabelPartOf]
	versionLabel := md.Labels[application.LabelVersion]

	// Special case: let a Helm chart define a missing name/version
	if nameLabel == "" || versionLabel == "" {
		if helmChartLabel := md.Labels[application.LabelHelmChart]; helmChartLabel != "" {
			chartName, chartVersion := splitHelmChart(helmChartLabel)
			if nameLabel == "" {
				nameLabel = chartName
			}
			if versionLabel == "" {
				versionLabel = chartVersion
			}
		}
	}

	// Build a pipeline of changes, the order in which we add to this list will
	// impact what the final document looks like
	var pp []yaml.Filter

	// Add Kubernetes metadata for the application
	pp = append(pp, yaml.Tee(yaml.SetField(yaml.APIVersionField, yaml.NewStringRNode("app.k8s.io/v1beta1"))))
	pp = append(pp, yaml.Tee(yaml.SetField(yaml.KindField, yaml.NewStringRNode("Application"))))
	if md.Namespace != "" {
		pp = append(pp, yaml.Tee(yaml.SetK8sNamespace(md.Namespace)))
	}

	// We need a name for the application: this is where the Application SIG
	// spec is a little confusing. They show examples where the application name
	// is something like "wordpress-01" implying that it is actually the
	// value of the "instance" label that corresponds to the application  name.
	switch {
	case instanceLabel != "" && nameLabel != "":
		pp = append(pp, yaml.Tee(yaml.SetK8sName(instanceLabel)))
		pp = append(pp, yaml.Tee(yaml.SetLabel(application.LabelName, nameLabel)))
	case instanceLabel != "":
		pp = append(pp, yaml.Tee(yaml.SetK8sName(instanceLabel)))
		pp = append(pp, yaml.Tee(yaml.SetLabel(application.LabelName, instanceLabel)))
	case nameLabel != "":
		pp = append(pp, yaml.Tee(yaml.SetK8sName(nameLabel)))
		pp = append(pp, yaml.Tee(yaml.SetLabel(application.LabelName, nameLabel)))
	default:
		// With no name we cannot create an application
		return nil, nil
	}

	// Add match labels for the current resource
	if nameLabel != "" {
		pp = append(pp, yaml.Tee(
			yaml.LookupCreate(yaml.MappingNode, "spec", "selector", "matchLabels"),
			yaml.Tee(yaml.SetField(nameLabelKey, yaml.NewStringRNode(nameLabel))),
		))
	}
	if instanceLabel != "" {
		pp = append(pp, yaml.Tee(
			yaml.LookupCreate(yaml.MappingNode, "spec", "selector", "matchLabels"),
			yaml.Tee(yaml.SetField(instanceLabelKey, yaml.NewStringRNode(instanceLabel))),
		))
	}

	// Add the current resource type as one of the component kinds supported by the application
	pp = append(pp, yaml.Tee(
		yaml.LookupCreate(yaml.SequenceNode, "spec", "componentKinds"),
		yaml.Append(&yaml.Node{Kind: yaml.MappingNode}),
		yaml.Tee(yaml.SetField("group", yaml.NewStringRNode(application.StripVersion(md.APIVersion)))),
		yaml.Tee(yaml.SetField("kind", yaml.NewStringRNode(md.Kind))),
	))

	// Consider part-of the application type, falling back on the name label
	// IFF we are using the instance label as the app name...
	appType := partOfLabel
	if appType == "" && instanceLabel != "" {
		appType = nameLabel
	}
	if appType != "" {
		pp = append(pp, yaml.Tee(
			yaml.LookupCreate(yaml.ScalarNode, "spec", "descriptor", "type"),
			yaml.Tee(yaml.Set(yaml.NewStringRNode(appType))),
		))
	}

	// Add the application version
	if versionLabel != "" {
		pp = append(pp, yaml.Tee(
			yaml.LookupCreate(yaml.ScalarNode, "spec", "descriptor", "version"),
			yaml.Tee(yaml.Set(yaml.NewStringRNode(versionLabel))),
		))
	}

	// Run the constructed pipeline over a new document
	return yaml.NewRNode(&yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{{Kind: yaml.MappingNode}},
	}).Pipe(pp...)
}

// splitHelmChart splits a Helm chart into it's name and version.
func splitHelmChart(chart string) (name string, version string) {
	name = regexp.MustCompile(`-(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)`).Split(chart, 2)[0]
	version = strings.TrimPrefix(strings.TrimPrefix(chart, name), "-")
	return
}
