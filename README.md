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

#### Handling Wildcards

`YAMLL` supports importing YAML files using wildcard patterns.

Filenames matching the pattern are excluded from visibility in the `tree`, `import`, and `build` commands. Instead, the data from all matching files is aggregated under the specified pattern.

For example, the pattern `##++internal/fixtures/*.test.yaml` might match files like `one.test.yaml`, `two.test.yaml`, and `three.test.yaml`. 

However, their individual filenames (`one.test.yaml`, `two.test.yaml`, and `three.test.yaml`) will not be displayed in the above commands.</br> 
Instead, their combined data will appear under the pattern `##++internal/fixtures/*.test.yaml`. (this makes it easier to manage the cyclic dependency and many others)

Following examples tries to illustrate all of them.

**Example** `root.yaml`:

```yaml
##++internal/fixtures/base.yaml
##++internal/fixtures/*.test.yaml
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

**Example** `one.test.yaml`:
```yaml
editor:
  - intellij
  - visual_code
```

**Example** `two.test.yaml`:
```yaml
movies:
  - animation
  - comedy
```

**Example** `three.test.yaml`:
```yaml
ott:
  - netflix
  - prime_video
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
# Source: internal/fixtures/*.test.yaml

editor:
   - intellij
   - visual_code
ott:
   - netflix
   - prime_video
movies:
   - animation
   - comedy
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
yamll tree -f import.yaml --output=json
yamll tree -f import.yaml --output=dot
yamll tree -f import.yaml --output=mermaid
```

`yamll tree` defaults to the current text tree output. Use `--output=json` for structured data, `--output=dot` for Graphviz, and `--output=mermaid` for Mermaid diagrams.

**Output**:
```sh
└── internal/fixtures/import.yaml
    ├── internal/fixtures/base.yaml
    │   └── internal/fixtures/base3.yaml
    ├── internal/fixtures/base2.yaml
    │   └── internal/fixtures/base3.yaml
    ├── internal/fixtures/*.test.yaml (3 files)
    │   ├── /Users/youruser/my-opensource/yamll/internal/fixtures/base.test.yaml
    │   ├── /Users/youruser/my-opensource/yamll/internal/fixtures/base2.test.yaml
    │   ├── /Users/youruser/my-opensource/yamll/internal/fixtures/base3.test.yaml
    │   ├── internal/fixtures/base4.yaml
    │   ├── internal/fixtures/*.testing.yaml (3 files)
    │   │   ├── /Users/youruser/my-opensource/yamll/internal/fixtures/one.testing.yaml
    │   │   ├── /Users/youruser/my-opensource/yamll/internal/fixtures/three.testing.yaml
    │   │   └── /Users/youruser/my-opensource/yamll/internal/fixtures/two.testing.yaml
    │   ├── internal/fixtures/base5.yaml
    │   └── internal/fixtures/base4.yaml
    ├── https://github.com/nikhilsbhat/yamll@main?path=internal/fixtures/base2.yaml
    │   └── internal/fixtures/base3.yaml
    └── http://localhost:3000/database
```

### Lint

Need a quick static check before building or tracing? `yamll lint` scans the dependency graph for common issues like duplicate keys, unresolved imports, unused imports, circular refs, invalid anchors, and conflicting merges.

**Example**:

```sh
yamll lint -f import.yaml
```

If issues are found, `yamll` prints them and exits with a non-zero status.

### Trace

Need to figure out where a specific rendered value came from?

`yamll trace` maps a generated YAML path back to its source file and line number (similar to source maps in compilers).

**Example**:

```sh
yamll trace internal/fixtures/import.yaml:base.movies
yamll trace -f internal/fixtures/import.yaml base.movies
yamll trace -f internal/fixtures/import.yaml workflow.dbname
```

**Output**:

```sh
origin: internal/fixtures/base5.yaml:2
```

### Lock File

Remote imports (git/URL) are powerful, but can drift over time. `yamll lock` generates a lock file with resolved commits and checksums, and subsequent commands will honor it by default.

More details: [LOCKFILE.md](docs/LOCKFILE.md)

**Example**:

```sh
yamll lock -f internal/fixtures/import.yaml
yamll import -f internal/fixtures/import.yaml
yamll build -f internal/fixtures/import.yaml
```

To ignore the lock file for a run:

```sh
yamll import -f internal/fixtures/import.yaml --no-lock
```
### Preventing Import Cycles

`yamll` detects and prevents import cycles. If an import cycle is detected, it will report an error and stop the merging
process.

## Documentation

Updated documentation on all available commands and flags can be
found [here](https://github.com/nikhilsbhat/yamll/blob/main/docs/doc/yamll.md).
