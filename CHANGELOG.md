# Changelog

This file is used to track unreleased changes, for a complete history, check the [releases page](https://github.com/carbonrelay/konjure/releases).

## Unreleased

### ✨ Added

* There is new consolidated Secret generator, it supports GCP Secret Manager and GPG encrypted secrets

### 🏗 Changed

* Upgrade Kustomize API to 0.6.0
* Upgrade Berglas to 0.5.3

### ⏳ Deprecated

* Berglas support is deprecated (in favor of Secret Manager) and will be removed in the next release
* Random secret generation is deprecated (in favor the new Secret generator) and will be removed in the next release

### 🛑 Removed

### 🐛 Fixed

### 🗝 Security

* When using GPG encrypted secrets, it is not recommended to use `pass:` pass phrases.
