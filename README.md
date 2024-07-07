# Yamll

[![Go Report Card](https://goreportcard.com/badge/github.com/nikhilsbhat/yamll)](https://goreportcard.com/report/github.com/nikhilsbhat/yamll)
[![shields](https://img.shields.io/badge/license-MIT-blue)](https://github.com/nikhilsbhat/yamll/blob/master/LICENSE)
[![shields](https://godoc.org/github.com/nikhilsbhat/yamll?status.svg)](https://godoc.org/github.com/nikhilsbhat/yamll)
[![shields](https://img.shields.io/github/v/tag/nikhilsbhat/yamll.svg)](https://github.com/nikhilsbhat/yamll/tags)
[![shields](https://img.shields.io/github/downloads/nikhilsbhat/yamll/total.svg)](https://github.com/nikhilsbhat/yamll/releases)

Yamll is a powerful tool for managing and merging multiple `YAML` files

## Introduction

This allows you to define dependencies on other `YAML` files, similar to how programming languages manage dependencies.

It ensures a single comprehensive YAML file by resolving interdependencies and preventing import cycles.

## Features

- Merge multiple `YAML` files into one
- Handle imports and dependencies seamlessly
- Detect and prevent import cycles
- Easy to use with clear error reporting
- Supports importing files from various source like `local path`, `GIT` repo and `HTTPS` source

### Authentication
- If authentication is required to connect to remote source defined. Creds can be passed as `environment` variable and `yamll` evaluates it
- In case of GIT, `yamll` supports both `ssh` and `http` based git URLs.
- All supported authentication parameters are defined [here](https://github.com/nikhilsbhat/yamll/blob/main/pkg/yamll/dependency.go#L31).

## Installation

* Recommend installing released versions. Release binaries are available on the [releases](https://github.com/nikhilsbhat/yamll/releases) page.

#### Homebrew

Install latest version on `yamll` on `macOS`

```shell
brew tap nikshilsbhat/stable git@github.com:nikhilsbhat/homebrew-stable.git
# for latest version
brew install nikshilsbhat/stable/yamll
# for specific version
brew install nikshilsbhat/stable/yamll@0.0.3
```

Check [repo](https://github.com/nikhilsbhat/homebrew-stable) for all available versions of the formula.

#### Docker

Latest version of docker images are published to [ghcr.io](https://github.com/nikhilsbhat/yamll/pkgs/container/yamll), all available images can be found there. </br>

```bash
docker pull ghcr.io/nikhilsbhat/yamll:latest
docker pull ghcr.io/nikhilsbhat/yamll:<github-release-tag>
```

#### Build from Source

1. Clone the repository:
    ```sh
    git clone https://github.com/nikhilsbhat/yamll.git
    cd yamll
    ```
2. Build the project:
    ```sh
    make local.build
    ```

## Usage

### Basic Usage

To merge multiple YAML files, simply specify the base YAML files as arguments:

```sh
yamll import -f import.yaml
```

### Handling Imports

YAML files can specify imports using the comments that starts with `##++`. `yamll` will resolve these imports and merge the contents.

It can construct the dependency tree and import them in the correct order, with each dependency able to have its own defined dependencies.

Following example tries to illustrate all of them.

**Example** `root.yaml`:

```yaml
##++internal/fixtures/base.yaml
##++git+https://github.com/nikhilsbhat/yamll@main?path=internal/fixtures/base2.yaml;{"user_name":"${GIT_USERNAME}","password":"${GITHUB_TOKEN}"}
##++http://localhost:3000/database.yaml

config2:
  test: val
  <<: *default

config3:
  - *default
  - *mysqldatabase

workflow: *mysqldatabase
```

**Example** `base.yaml`:
```yaml
default: &default
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: base-config
  data:
    key1: value1
    key2: value2
config1: *default
```

**Example** `base2.yaml` retrieved from `GIT` source:
```yaml
names:
   - john doe
   - dexter
```

**Example** `base3.yaml`:
```yaml
organizations:
  - thoughtworks
  - google
  - microsoft
```

`database.yaml` retrieved from `URL` source:
```yaml
mysqldatabase: &mysqldatabase
  hostname: localhost
  port: 3012
  username: root
  password: root
```

Importing `root.yaml` should generate final yaml file as below

```yaml
---
# Source: internal/fixtures/base3.yaml
organizations:
  - thoughtworks
  - google
  - microsoft
---
# Source: internal/fixtures/base.yaml
default: &default
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: base-config
  data:
    key1: value1
    key2: value2
config1: *default
---
# Source: https://github.com/nikhilsbhat/yamll@main?path=internal/fixtures/base2.yaml
names:
  - john doe
  - dexter
---
# Source: http://localhost:3000/database.yaml
mysqldatabase: &mysqldatabase
  hostname: localhost
  port: 3012
  username: root
  password: root
---
# Source: internal/fixtures/import.yaml
config2:
  test: val
  <<: *default
config3:
  - *default
  - *mysqldatabase
workflow: *mysqldatabase
```

### Dependency Tree

Want to see all your dependencies in a tree format? This `yamll` tool supports that too.

Using `yaml tree` will print dependencies just like the Linux `tree command`.

**Example**:

```sh
yamll tree -f import.yaml
```

**Output**:
```sh
└── internal/fixtures/import.yaml
    ├── internal/fixtures/base.yaml
    │   └── internal/fixtures/base3.yaml
    ├── internal/fixtures/base2.yaml
    │   └── internal/fixtures/base3.yaml
    ├── https://github.com/nikhilsbhat/yamll@main?path=internal/fixtures/base2.yaml
    │   └── internal/fixtures/base3.yaml
    └── http://localhost:3000/database.yaml
```
### Preventing Import Cycles

`yamll` detects and prevents import cycles. If an import cycle is detected, it will report an error and stop the merging
process.

## Documentation

Updated documentation on all available commands and flags can be
found [here](https://github.com/nikhilsbhat/yamll/blob/main/docs/doc/yamll.md).