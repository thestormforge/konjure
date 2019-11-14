# Berglas Generator

The Berglas Generator is used to generate Kubernetes secrets from secrets stored using [Berglas](https://github.com/GoogleCloudPlatform/berglas).

## Prerequisites

For this example, you will need Konjure and Kustomize. You can also use Konjure without Kustomize if you like.

```sh
which konjure
which kustomize
konjure kustomize init BerglasGenerator

DEMO_HOME=$(mktemp -d)
mkdir -p "$DEMO_HOME"
```

## Configuration

Create a configuration file for the Berglas Generator, in this case we will generate a secret with two entries and we will disable the name suffix using standard Kustomize generator options:

```sh
cat <<'EOF' >"$DEMO_HOME/berglas.yaml"
apiVersion: konjure.carbonrelay.com/v1beta1
kind: BerglasGenerator
metadata:
  name: ignored

name: my-secret
refs:
  - berglas://berglas-konjure-test-secrets/api-key
  - berglas://berglas-konjure-test-secrets/tls-key?destination=tempfile

generatorOptions:
  disableNameSuffixHash: true
EOF
```

## Kustomize

Create a kustomization the uses the Berglas Generator configuration, for this example we are only including the resources generated from the Berglas secrets, however you can use the generator as part of a larger Kustomization as well.

```sh
cat <<'EOF' >"$DEMO_HOME/kustomization.yaml"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

generators:
  - berglas.yaml
EOF
```

## Build

Finally, build the manifest.

```sh
kustomize build "$DEMO_HOME" --enable_alpha_plugins
```

## Clean Up

Remove your demo workspace:

```sh
rm -rf "$DEMO_HOME"
```
