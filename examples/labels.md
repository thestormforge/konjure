# Label Transformer

The Label Transformer is an alternate to the built-in label transformer that avoids the creation of selectors in certain scenarios.

Creating missing selectors for the purpose of adding labels creates an application that is fundamentally different from the original. For example, given the following manifest describing two deployments:

```sh
DEMO_HOME=$(mktemp -d)
mkdir -p "$DEMO_HOME"
cat <<'EOF' >"$DEMO_HOME/deployments.yaml"
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: example-1
spec:
  template:
    metadata:
      labels:
        app: example-1
    spec:
      containers:
        - name: echo
          image: busybox
          command: "echo 'Hello, World!'"
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: example-2
spec:
  template:
    metadata:
      labels:
        app: example-2
    spec:
      containers:
        - name: echo
          image: busybox
          command: "echo 'Goodbye, World!'"
EOF
```

Transforming this with Kustomize (at least up to v3.4.0) `commonLabels` will create the "missing" selector; e.g. adding the label `foo=bar` will result in:

```sh
cat <<'EOF' >"$DEMO_HOME/deployments-bad-transformation.yaml"
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: example-1
  labels:
    foo: bar
spec:
  selector:
    foo: bar
  template:
    metadata:
      labels:
        app: example-1
        foo: bar
    spec:
      containers:
        - name: echo
          image: busybox
          command: "echo 'Hello, World!'"
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: example-2
  labels:
    foo: bar
spec:
  selector:
    foo: bar
  template:
    metadata:
      labels:
        app: example-2
        foo: bar
    spec:
      containers:
        - name: echo
          image: busybox
          command: "echo 'Goodbye, World!'"
EOF
```

This is clearly incorrect because both the `example-1` and `example-2` deployments now match the same pods.

Note the use of `extensions/v1beta1` deployments (a version which is [unsupported as of Kubernetes 1.16](https://kubernetes.io/blog/2019/07/18/api-deprecations-in-1-16/)): the selector isn't required and the default behavior is to use the template labels for the selector match labels (this behavior is documented on the replica set). In `apps/v1` the deployment selector is required and can match labels or expressions: this makes creation of the selector match labels safe (it either already exists or, if created, will be logically ANDed to an existing condition set).

However, the default meaning of an absent selector is not limited to (as of now) unsupported APIs: for example, `batch/v1` jobs also apply semantic meaning to "empty or non-specified selectors" (although Kustomize common labels will not create selectors on jobs).

## Prerequisites

For this example, you will need Konjure and Kustomize installed on your `PATH`. If you plan to use Konjure without Kustomize you can.

```sh
which konjure
which kustomize
konjure kustomize init LabelTransformer
```

## Configuration

Create a configuration file for the Label Transformer, the format is the same as [configuration for the built-in plugin](https://github.com/kubernetes-sigs/kustomize/blob/master/docs/plugins/builtins.md#usage-via-plugin-3) except for the `apiVersion`.

```sh
cat <<'EOF' >"$DEMO_HOME/labels.yaml"
apiVersion: konjure.carbonrelay.com/v1beta1
kind: LabelTransformer
metadata:
  name: ignored

labels:
  foo: bar
EOF
```

## Kustomize

Create a kustomization that uses the Label Transformer configuration, for this example we will transform the example `extensions/v1beta1` deployments.

```sh
cat <<'EOF' >"$DEMO_HOME/kustomization.yaml"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - deployments.yaml

transformers:
  - labels.yaml
EOF
```

## Build

Finally, build the manifests. Notice that selectors are not created on the legacy types.

```sh
kustomize build "$DEMO_HOME" --enable_alpha_plugins > "$DEMO_HOME/kustomized.yaml"
[ ! $(grep -q 'selector:' "$DEMO_HOME/kustomized.yaml" && echo $?) ]
```

## Clean Up

Remove your demo workspace:

```sh
rm -rf "$DEMO_HOME"
```

