# 🧙‍ Konjure

![](https://github.com/thestormforge/konjure/workflows/Master/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/thestormforge/konjure)](https://goreportcard.com/report/github.com/thestormforge/konjure)

Konjure generates and transforms Kubernetes resource definitions. It can be used as a standalone utility or can be integrated into your GitOps workflows.

## Installation

Download the [latest binary](https://github.com/thestormforge/konjure/releases/latest) for your platform and put it on your path:

```sh
os=linux # Or 'darwin'
curl -s https://api.github.com/repos/thestormforge/konjure/releases/latest |\
  grep browser_download_url | grep ${os:-linux} | cut -d '"' -f 4 |\
  xargs curl -L | tar xz
sudo mv konjure /usr/local/bin/
```

## Usage

Konjure can be used to aggregate or generate Kubernetes manifests from a number of different sources. Simply invoke Konjure with a list of sources.

### Konjure Sources

Konjure supports pulling resources from the following sources:

* Local files or directories
* Git repositories
* HTTP resources
* Helm charts (via `helm template`)
* Kustomize
* Kubernetes
* Jsonnet

Konjure also has its own resource generators:

* Secret generator

Some sources can be specified using a URL: file system paths, HTTP URLs, and Git repository URLs can all be entered directly. Helm chart URLs can also be used when prefixed with `helm::`.

### Konjure Resources

Konjure defines several Kubernetes-like resources which will be expanded in place during execution. For example, if Konjure encounters a resource with the `apiVersion: konjure.stormforge.io/v1beta2` and the `kind: File` it will be replaced with the manifests found in the named file.

The current (and evolving) definitions can be found in the [API source](api/core/v1beta2/types.go).