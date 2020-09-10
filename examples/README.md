# Konjure Examples

## Example Usage

* [Helm Generator](helm-generator.md)
* [Jsonnet Generator](jsonnet-generator.md)
* [Label Transformer](labels-transformer.md)
* [Secret Generator](secret-generator.md)
* [Secret Generator - GPG](secret-generator-gpg.md)
* [Version Transformer](version-transformer.md)

## Kustomize vs. CLI

These examples all use Konjure as a Kustomize executable plugin. While this is preferred way to use Konjure, all of the generators and transformers can also be used without Kustomize through direct invocation of the `konjure` binary.

For example, the Helm Generator example can be run directly from the CLI using:

```sh
STABLE_URL=https://kubernetes-charts.storage.googleapis.com/
konjure helm --name elasticsearch --version 1.31.1 --repo $STABLE_URL --set data.replicas=3 elasticsearch
```
