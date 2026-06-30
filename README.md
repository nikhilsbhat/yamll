# Yamll

[![Go Report Card](https://goreportcard.com/badge/github.com/nikhilsbhat/yamll)](https://goreportcard.com/report/github.com/nikhilsbhat/yamll)
[![shields](https://img.shields.io/badge/license-MIT-blue)](https://github.com/nikhilsbhat/yamll/blob/master/LICENSE)
[![shields](https://godoc.org/github.com/nikhilsbhat/yamll?status.svg)](https://godoc.org/github.com/nikhilsbhat/yamll)
[![shields](https://img.shields.io/github/v/tag/nikhilsbhat/yamll.svg)](https://github.com/nikhilsbhat/yamll/tags)
[![shields](https://img.shields.io/github/downloads/nikhilsbhat/yamll/total.svg)](https://github.com/nikhilsbhat/yamll/releases)

Yamll keeps shared YAML from turning into copy-paste drift.
It resolves imports, expands anchors, and lets you trace every field back to where it came from.

![Yamll terminal demo](docs/assets/yamll-terminal-demo.gif)

## Why Yamll

When the same config gets repeated across environments, changes start to drift.
Yamll lets you define shared YAML once, import it from files, Git, HTTP, or OCI, and inspect the graph when something breaks.

## Features

- Merge multiple `YAML` files into one output
- Resolve imports across local files, Git, HTTP, and OCI sources
- Catch import cycles, duplicate keys, invalid anchors, and merge issues early
- Trace rendered values back to their source file and line
- Lock remote imports for reproducible builds

## Quick Demo

Start with one root file and one shared base:

```yaml
# root.yaml
##++internal/fixtures/base.yaml

app:
  name: api
  <<: *default
```

```yaml
# internal/fixtures/base.yaml
default: &default
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: base-config
```

Build it:

```sh
yamll import -f root.yaml
```

Then trace a value back to its origin:

```sh
yamll trace root.yaml:app.metadata.name
```

### Authentication
- Pass credentials through `environment` variables and `yamll` will resolve them at runtime
- Git imports support both `ssh` and `http` URLs
- OCI imports work with registry-hosted config bundles and artifacts
- All supported authentication parameters are defined [here](https://github.com/nikhilsbhat/yamll/blob/main/pkg/yamll/dependency.go#L34)

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

Point `yamll` at your root YAML file and it pulls the graph together:

```sh
yamll import -f import.yaml
```

### Handling Imports

Imports live in comments that start with `##++`. `yamll` resolves them, walks the dependency tree, and merges everything in the right order.

#### Handling Wildcards

Wildcard imports keep noisy file lists out of the way.

Filenames matching the pattern stay hidden in `tree`, `import`, and `build`. Their data is folded under the pattern itself.

For example, `##++internal/fixtures/*.test.yaml` might match `one.test.yaml`, `two.test.yaml`, and `three.test.yaml`.

Their names disappear from the command output, and the combined content appears under the pattern import. It keeps cyclic graphs and large fixture sets readable.

The examples below show the common cases.

**Example** `root.yaml`:

```yaml
##++internal/fixtures/base.yaml
##++internal/fixtures/*.test.yaml
##++git+https://github.com/nikhilsbhat/yamll@main?path=internal/fixtures/base2.yaml;{"user_name":"${GIT_USERNAME}","password":"${GITHUB_TOKEN}"}
##++oci://ghcr.io/company/platform-config:v1
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

Need the graph? `yamll tree` prints it like a filesystem tree.

**Example**:

```sh
yamll tree -f import.yaml
yamll tree -f import.yaml --output=json
yamll tree -f import.yaml --output=dot
yamll tree -f import.yaml --output=mermaid
```

`yamll tree` defaults to the text tree. Use `--output=json` for structured data, `--output=dot` for Graphviz, and `--output=mermaid` for Mermaid.

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

### Impact Analysis

Need the blast radius? `yamll impact` walks the graph in reverse and shows every downstream file that depends on the target.

**Example**:

```sh
yamll impact common.yaml
yamll impact -f internal/fixtures/import.yaml internal/fixtures/base.yaml
```

**Output**:

```text
Affected files:
  api.yaml
  ingress.yaml
  web.yaml
  jobs.yaml

Total downstream dependencies: 17
```

### Lint

Want a fast sanity check? `yamll lint` scans the graph for duplicate keys, unresolved imports, unused imports, circular refs, invalid anchors, and conflicting merges.

**Example**:

```sh
yamll lint -f import.yaml
```

If issues are found, `yamll` prints them and exits with a non-zero status.

### Trace

Need the origin of a rendered value?

`yamll trace` maps a generated YAML path back to the source file and line number, compiler-style.

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

Remote imports are powerful, but drift. `yamll lock` records resolved commits and checksums, and future runs fail if the fetched content no longer matches the lock.

More details: [LOCKFILE.md](docs/LOCKFILE.md)

**Example**:

```sh
yamll lock -f internal/fixtures/import.yaml
yamll import -f internal/fixtures/import.yaml
yamll build -f internal/fixtures/import.yaml
```

To verify the lock without rendering output:

```sh
yamll lock verify -f internal/fixtures/import.yaml
```

To explain which roots pull in a dependency:

```sh
yamll lock explain internal/fixtures/base.yaml -f app.yaml -f jobs.yaml
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
