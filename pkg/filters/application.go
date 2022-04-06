package filters

import (
	"github.com/thestormforge/konjure/pkg/filters/internal"
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

	apps := make(map[yaml.NameMeta]*internal.ApplicationNode)
	var err error
	var scannedAppLabels bool

IndexApps:

	// Index the existing applications
	nodes, err = internal.IndexApplications(nodes, apps)
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

func (f *ApplicationFilter) appFromLabels(n *yaml.RNode) (*yaml.RNode, error) {
	md, err := n.GetMeta()
	if err != nil {
		return nil, err
	}

	nameLabelKey := f.ApplicationNameLabel
	if nameLabelKey == "" {
		nameLabelKey = "app.kubernetes.io/name"
	}

	instanceLabelKey := f.ApplicationInstanceLabel
	if instanceLabelKey == "" {
		instanceLabelKey = "app.kubernetes.io/instance"
	}

	nameLabel := md.Labels[nameLabelKey]
	instanceLabel := md.Labels[instanceLabelKey]
	partOfLabel := md.Labels["app.kubernetes.io/part-of"]
	versionLabel := md.Labels["app.kubernetes.io/version"]

	// Special case: let a Helm chart define a missing name/version
	if nameLabel == "" || versionLabel == "" {
		if helmChartLabel := md.Labels["helm.sh/chart"]; helmChartLabel != "" {
			chartName, chartVersion := internal.SplitHelmChart(helmChartLabel)
			if nameLabel == "" {
				nameLabel = chartName
			}
			if versionLabel == "" {
				versionLabel = chartVersion
			}
		}
	}

	// We need a name for the application: this is where the Application SIG
	// spec is a little confusing. They show examples where the application name
	// is something like "wordpress-01" implying that it is actually the
	// value of the "instance" label that corresponds to the application name.
	// For this implementation, we'll try the instance label first, and fallback
	// on the name label; but at least one of them needs to be non-empty.
	name := instanceLabel
	if name == "" {
		name = nameLabel
	}
	if name == "" {
		return nil, nil
	}

	var pp []yaml.Filter

	// Add Kubernetes metadata for the application
	pp = append(pp, yaml.Tee(yaml.SetField(yaml.APIVersionField, yaml.NewStringRNode("app.k8s.io/v1beta1"))))
	pp = append(pp, yaml.Tee(yaml.SetField(yaml.KindField, yaml.NewStringRNode("Application"))))
	if md.Namespace != "" {
		pp = append(pp, yaml.Tee(yaml.SetK8sNamespace(md.Namespace)))
	}
	pp = append(pp, yaml.Tee(yaml.SetK8sName(name)))
	pp = append(pp, yaml.Tee(yaml.SetLabel("app.kubernetes.io/name", name))) // As seen in the Application SIG examples

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
	gk := internal.GroupKindFromType(md.TypeMeta)
	pp = append(pp, yaml.Tee(
		yaml.LookupCreate(yaml.SequenceNode, "spec", "componentKinds"),
		yaml.Append(&yaml.Node{Kind: yaml.MappingNode}),
		yaml.Tee(yaml.SetField("group", yaml.NewStringRNode(gk.Group))),
		yaml.Tee(yaml.SetField("kind", yaml.NewStringRNode(gk.Kind))),
	))

	// Consider part-of the application type, falling back on the name label IFF we are using the instance label as the app name...
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
	return yaml.NewRNode(&yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}).Pipe(pp...)
}
