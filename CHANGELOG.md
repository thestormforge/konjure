# Changelog

This file is used to track unreleased changes, for a complete history, check the [releases page](https://github.com/carbonrelay/konjure/releases).

## Unreleased

### ✨ Added

* Added a `konjure cat` command to concatenate and print manifest files.

### 🏗 Changed

* Upgraded Kustomize API to 0.4.1.
* The Helm generator accepts an explicit namespace for rendering templates.

### ⏳ Deprecated

### 🛑 Removed

* The Helm generator CLI now uses `-n` to configure the namespace instead of the release name; the long form of the argument (`--name`) remains unchanged.

### 🐛 Fixed

* Fixed an issue where the `konjure env` command wasn't working with secrets.

### 🗝 Security
