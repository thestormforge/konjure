package application

// See: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
// See: https://helm.sh/docs/chart_best_practices/labels/

const (
	// LabelName is the recommended label for the name of the
	// application. For example, `mysql`.
	LabelName = "app.kubernetes.io/name"
	// LabelInstance is the recommended label for a unique name
	// identifying the instance of an application. For example, `mysql-abcxzy`.
	LabelInstance = "app.kubernetes.io/instance"
	// LabelVersion is the recommended label for the current version
	// of the application. For example, `5.7.21`.
	LabelVersion = "app.kubernetes.io/version"
	// LabelComponent is the recommended label for the component
	// within the architecture. For example, `database`.
	LabelComponent = "app.kubernetes.io/component"
	// LabelPartOf is the recommended label for the name of a higher
	// level application this one is part of. For example, `wordpress`.
	LabelPartOf = "app.kubernetes.io/part-of"
	// LabelManagedBy is the recommended label for the tool being
	// used to manage the operation of an application. For example, `helm`.
	LabelManagedBy = "app.kubernetes.io/managed-by"
	// LabelCreatedBy is the recommended label for the controller/user
	// who created this resource. For example, `controller-manager`.
	LabelCreatedBy = "app.kubernetes.io/created-by"
	// LabelHelmChart is the recommended label for the chart name and version.
	// For example, `{{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}`.
	LabelHelmChart = "helm.sh/chart"
)
