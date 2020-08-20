# Secret Generator - GPG

The Secret Generator can decrypt GPG encrypted secrets during generation. Secret keys with a `.gpg` file extension will be decrypted.

## Prerequisites

For this example, you will need GPG, Konjure, and Kustomize installed on your `PATH`. If you plan to Konjure without Kustomize you can.

```sh
which gpg
which konjure
which kustomize
konjure kustomize init SecretGenerator

DEMO_HOME=$(mktemp -d)
mkdir -p "$DEMO_HOME"
```

## Configuration

Create a configuration file for the Secret Generator, in this case we will generate a random UUID and a random password.

```sh
export LARGE_SECRET_PASSPHRASE=blahblahblah
echo '{"test":true}' | gpg --batch --symmetric --cipher-algo AES256 --output "$DEMO_HOME/my_secret.json.gpg" --passphrase "$LARGE_SECRET_PASSPHRASE"

cat <<'EOF' >"$DEMO_HOME/gpg.yaml"
apiVersion: konjure.carbonrelay.com/v1beta1
kind: SecretGenerator
metadata:
  name: ignored

name: gpg-secret
passPhrases:
  my_secret.json: 'env:LARGE_SECRET_PASSPHRASE'
files:
  - my_secret.json.gpg
EOF
```

## Kustomize

Create a kustomization the uses the Random Generator configuration, for this example we are only including the secret.

```sh
cat <<'EOF' >"$DEMO_HOME/kustomization.yaml"
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

generators:
  - gpg.yaml
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
