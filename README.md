# 🧙‍ Konjure

![](https://github.com/carbonrelay/konjure/workflows/Main%20workflow/badge.svg)

Konjure generates and transforms Kubernetes resource definitions. It can be used as a standalone utility or can be integrated into your Kustomize workflow.

## Installation

Download the latest binary for your platform, make it executable and put it on your path:

```sh
os=linux # Or 'darwin'
curl -s https://api.github.com/repos/carbonrelay/konjure/releases/latest |\
  jq -r ".assets[] | select(.name | contains(\"${os:-linux}\")) | .browser_download_url" |\
  xargs curl -L -o konjure
chmod +x konjure
sudo mv konjure /usr/local/bin/
```

For Kustomization integration, you can run:

```sh
konjure kustomize init
```

## Usage

Konjure can be used as a standalone tool by invoking the `konjure` tool directly; all Konjure commands can also be accessed as Kustomize plugins.

## Getting Help

Run `konjure --help` to get a list of the currently supported generators and transformations.