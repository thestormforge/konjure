# 🧙‍ Konjure

![](https://github.com/thestormforge/konjure/workflows/Main/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/thestormforge/konjure)](https://goreportcard.com/report/github.com/thestormforge/konjure)

Konjure generates and transforms Kubernetes resource definitions. It can be used as a standalone utility or can be integrated into your GitOps workflows.

## Installation

### Binaries

Each [release](https://github.com/thestormforge/konjure/releases/) includes binaries for multiple OS/Arch combinations which can be manually downloaded and installed.

1. Download the appropriate binary for your platform from the [releases page](https://github.com/thestormforge/konjure/releases/).
2. Unpack it, for example: `tar -xzf konjure-linux-amd64.tar.gz`
3. Move the `konjure` binary to the desired location, for example: `/usr/local/bin/konjure`

### Homebrew (macOS)

Install via the StormForge Tap:

```shell
brew install thestormforge/tap/konjure
```

## Usage

Konjure can be used to aggregate or generate Kubernetes manifests from a number of different sources. Simply invoke Konjure with a list of sources.

In the simplest form, Konjure acts like `cat` for Kubernetes manifests, for example the following will emit a YAML document stream with the resources from two files:

```shell
konjure service.yaml deployment.yaml
```

Konjure can convert the resources into [NDJSON](http://ndjson.org/) (Newline Delimited JSON) using the `--output ndjson` option (for example, to pipe into [`jq -s`](https://stedolan.github.io/jq/)). It can also apply some basic filters such as `--format` (for consistent field ordering and YAML formatting conventions) or `--keep-comments=false` (to strip comments); use `konjure --help` to see additional options.

### Konjure Sources

In addition to the local file system, Konjure supports pulling resources from the following sources:

* Local directories
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

Konjure defines several Kubernetes-like resources which will be expanded in place during execution. For example, if Konjure encounters a resource with the `apiVersion: konjure.stormforge.io/v1beta2` and the `kind: File` it will be replaced with the manifests found in the named file. Konjure resources are expanded iteratively, by using the `--depth N` option you can limit the number of expansions (for example, `--depth 0` is useful for creating a Konjure resource equivalent to the current invocation of Konjure).

The current (and evolving) definitions can be found in the [API source](pkg/api/core/v1beta2/types.go).
