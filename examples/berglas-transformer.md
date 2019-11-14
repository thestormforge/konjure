# Berglas Transformer

The Berglas Transformer will transform pod templates and adjust their manifest to pull secrets from Berglas. It is based on the [Kubernetes Mutating Webhook](https://github.com/GoogleCloudPlatform/berglas/tree/master/examples/kubernetes), except that instead of inspecting pods at admission, it looks for pod templates during kustomization. The transformation is applied to registered resource kinds whose `spec/template` field is a pod template specification.

Since the transformation is done at kustomization, there is no need to deploy the mutating webhook. However, your pods will still need [permission](https://github.com/GoogleCloudPlatform/berglas/tree/master/examples/kubernetes#permissions) to access the Berglas secrets; please refer to the Berglas documentation for more details.

## Prerequisites

For this example, you will need Konjure and Kustomize. You can also use Konjure without Kustomize if you like.

```sh
which konjure
which kustomize
konjure kustomize init BerglasTransformer

DEMO_HOME=$(mktemp -d)
mkdir -p "$DEMO_HOME"
```

## Configuration

Create a configuration file for the Berglas Transformer, only the metadata is required as the transformation is not configurable.

```sh
cat <<'EOF' >"$DEMO_HOME/berglas.yaml"
apiVersion: konjure.carbonrelay.com/v1beta1
kind: BerglasTransformer
metadata:
  name: ignored
EOF
```

## Kustomize

Create a kustomization the uses the Berglas Transformer configuration, for this example we will also be using the sample resources from the Berglas repository.

```sh
pushd "$DEMO_HOME"
curl -LO https://raw.githubusercontent.com/GoogleCloudPlatform/berglas/master/examples/kubernetes/deploy/sample.yaml
kustomize create --autodetect
konjure kustomize edit add transformer berglas.yaml
popd
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
