# 🧙‍ Konjure

![](https://github.com/thestormforge/konjure/workflows/Master/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/thestormforge/konjure)](https://goreportcard.com/report/github.com/thestormforge/konjure)

Konjure generates and transforms Kubernetes resource definitions. It can be used as a standalone utility or can be integrated into your Kustomize workflow.

## Installation

Download the [latest binary](https://github.com/thestormforge/konjure/releases/latest) for your platform and put it on your path:

```sh
os=linux # Or 'darwin'
curl -s https://api.github.com/repos/thestormforge/konjure/releases/latest |\
  grep browser_download_url | grep ${os:-linux} | cut -d '"' -f 4 |\
  xargs curl -L | tar xz
sudo mv konjure /usr/local/bin/
```

To enable the Kustomization integration, you can also run:

```sh
konjure kustomize init
```

## Usage

Konjure can be used as a standalone tool by invoking the `konjure` tool directly; all Konjure commands can also be accessed as Kustomize plugins, see the [examples](examples/) for more information.

## Getting Help

Run `konjure --help` to get a list of the currently supported generators and transformations.