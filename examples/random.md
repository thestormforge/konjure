# Random Generator

The Random Generator is used to create secrets containing random values.

## Prerequisites

For this example, you will need Konjure and Kustomize installed on your `PATH`. If you plan to Konjure without Kustomize you can.

```sh
which konjure
which kustomize

DEMO_HOME=$(mktemp -d)
mkdir -p "$DEMO_HOME"
```

## Configuration

Create a configuration file for the Random Generator, in this case we will generate two random passwords.

```sh
cat <<'EOF' >"$DEMO_HOME/random.yaml"
apiVersion: konjure.carbonrelay.com/v1beta1
kind: RandomGenerator
metadata:
  name: ignored

name: random-secret
passwords:
  - key: client_id
    length: 32
  - key: client_secret
    length: 64
EOF
```

## Kustomize

Create a kustomization the uses the Random Generator configuration, for this example we are only including the secret.

```sh
cat <<'EOF' >"$DEMO_HOME/kustomization.yaml"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

generators:
  - random.yaml
EOF
```

## Build

Finally, build the manifests. Notice that there are no test hook pods present in the output as they are filtered out by default.

```sh
kustomize build "$DEMO_HOME" --enable_alpha_plugins
```

## Clean Up

Remove your demo workspace:

```sh
rm -rf "$DEMO_HOME"
```
