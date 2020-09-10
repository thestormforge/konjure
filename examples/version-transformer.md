# Version Transformer

The Version Transformer is an opinionated transformer to simplify managing versioned resources. The transformer will attempt to extract a version number from the reference of a Git based resource in the configuration. All resources originating from that resource specification will be given a `app.kubernetes.io/version` label and, if any images have names matching the repository slug (i.e. the repository owner and name), their tags will be updated as well.

## Prerequisites

For this example, you will need Konjure and Kustomize installed on your `PATH`. If you plan to use Konjure without Kustomize you can.

```sh
which konjure
which kustomize
konjure kustomize init VersionTransformer
```

We will also need a tagged Git repository with some resources to reference. Note that in this example we _will not_ be having our images tagged, only labels will be applied.

```sh
DEMO_HOME=$(mktemp -d)
mkdir -p "$DEMO_HOME"
git init "$DEMO_HOME/project"

cat <<'EOF' >"$DEMO_HOME/project/kustomization.yaml"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- deployment.yaml
EOF
git -C "$DEMO_HOME/project" add "kustomization.yaml"

cat <<'EOF' >"$DEMO_HOME/project/deployment.yaml"
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-1
spec:
  selector:
    name: example-1
  template:
    metadata:
      labels:
        app: example-1
    spec:
      containers:
      - name: echo
        image: busybox
        command: "echo 'Hello, World!'"
EOF
git -C "$DEMO_HOME/project" add "deployment.yaml"

git -C "$DEMO_HOME/project" commit -m 'Initial (and only) commit'
git -C "$DEMO_HOME/project" tag 'v1.0.0'
```

## Configuration

Create a configuration file for the Version Transformer.

```sh
cat <<'EOF' >"$DEMO_HOME/version.yaml"
apiVersion: konjure.carbonrelay.com/v1beta1
kind: VersionTransformer
metadata:
  name: ignored
EOF
```

## Kustomize

Create a kustomization that uses the Version Transformer configuration.

```sh
cat <<EOF >"$DEMO_HOME/kustomization.yaml"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
- git::file://$DEMO_HOME/project?ref=v1.0.0

transformers:
- version.yaml
EOF
```

## Build

Finally, build the manifests. The version number from the Git reference will be applied as a label.

```sh
kustomize build "$DEMO_HOME" --enable_alpha_plugins > "$DEMO_HOME/kustomized.yaml"
[ $(grep -q 'app.kubernetes.io/version: v1.0.0' "$DEMO_HOME/kustomized.yaml" && echo $?) ]
```

## Clean Up

Remove your demo workspace:

```sh
rm -rf "$DEMO_HOME"
```

