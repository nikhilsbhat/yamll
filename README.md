# Yamll

[![shields](https://img.shields.io/badge/license-MIT-blue)](https://github.com/nikhilsbhat/yamll/blob/main/LICENSE)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/nikhilsbhat/yamll.svg)](https://pkg.go.dev/github.com/nikhilsbhat/yamll)
[![GitHub Release](https://img.shields.io/github/v/tag/nikhilsbhat/yamll.svg)](https://github.com/nikhilsbhat/yamll/releases)
[![Downloads](https://img.shields.io/github/downloads/nikhilsbhat/yamll/total.svg)](https://github.com/nikhilsbhat/yamll/releases)

Stop copying the same YAML across repositories.

**Yamll** is the dependency manager for YAML.

Import, merge, trace, lint, and lock YAML configuration across local files, Git repositories, OCI registries, and HTTP sources.

**Perfect for:**
- Kubernetes manifests
- Helm values
- GitHub Actions
- GoCD pipelines
- Docker Compose
- ArgoCD / GitOps
- CI/CD pipelines
- Shared application configuration

![Yamll terminal demo](docs/assets/yamll-terminal-demo.gif)

## Highlights

- 🚀 Import YAML from local files, Git, OCI, and HTTP
- 🔍 Trace rendered values back to their source
- 🌳 Visualize dependency graphs
- 🔒 Lock remote imports for reproducible builds
- ⚡  Detect configuration problems before deployment

## Why Yamll

As YAML grows across repositories and environments, duplication becomes difficult to manage. A small change often means updating dozens of files manually, leading to configuration drift and hard-to-debug deployments.

Yamll lets you treat YAML like source code:

- Import configuration from local files, Git repositories, OCI registries, and HTTP endpoints
- Compose reusable configuration modules
- Visualize dependency graphs
- Trace rendered values back to their origin
- Detect configuration problems before deployment
- Lock remote dependencies for reproducible builds

Instead of maintaining large monolithic YAML files or copying snippets between repositories, Yamll helps you build modular, reusable configuration.

## When should I use Yamll?

Use Yamll when you need to:

- Reuse the same YAML safely across services, repositories, or environments
- Understand where a rendered value came from and why it won
- Keep remote YAML imports reproducible with lock files and checksums
- Manage generic YAML, not only Kubernetes-specific manifests

## Quick Demo

Start with one root file and one shared base:

### root.yaml

```yaml
# root.yaml
##++internal/fixtures/base.yaml

app:
  name: api
  <<: *default
```

### base.yaml

```yaml
# internal/fixtures/base.yaml
default: &default
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: base-config
```

Render the final YAML.

```sh
yamll build -f root.yaml
```

Output:

```yaml
app:
  name: api
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: base-config
```

Need to know where a value came from?

```bash
yamll trace root.yaml:app.metadata.name
```

Need to know why one value won over another?

```bash
yamll explain root.yaml:app.metadata.name
```

## Comparison

| Capability                | Yamll | yq |  Helm   | Kustomize |
|---------------------------|:-----:|:--:|:-------:|:---------:|
| Import arbitrary YAML     |   ✅   | ❌  | Partial |     ❌     |
| Merge YAML                |   ✅   | ✅  |    ✅    |     ✅     |
| Dependency graph          |   ✅   | ❌  |    ❌    |     ❌     |
| Trace rendered values     |   ✅   | ❌  |    ❌    |     ❌     |
| Explain merge resolution  |   ✅   | ❌  |    ❌    |     ❌     |
| Lock remote dependencies  |   ✅   | ❌  |    ❌    |     ❌     |
| Generic YAML              |   ✅   | ✅  |    ❌    |     ❌     |
| Kubernetes-specific       |   ❌   | ❌  |    ✅    |     ✅     |

## Installation

* Recommend installing released versions. Release binaries are available on the [releases](https://github.com/nikhilsbhat/yamll/releases) page.

#### Homebrew

Install the latest version of `yamll` on `macOS`

```shell
brew tap nikhilsbhat/stable git@github.com:nikhilsbhat/homebrew-stable.git
# for latest version
brew install nikhilsbhat/stable/yamll
# for specific version
brew install nikhilsbhat/stable/yamll@0.0.3
```

Check [repo](https://github.com/nikhilsbhat/homebrew-stable) for all available versions of the formula.

#### Docker

Docker images are published to [ghcr.io](https://github.com/nikhilsbhat/yamll/pkgs/container/yamll), and all available tags can be found there.

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

## Typical Use Cases

Yamll is particularly useful for:

- Splitting large Kubernetes manifests into reusable modules
- Managing layered Helm values
- Organizing GitHub Actions workflows
- Reusing GoCD pipeline fragments
- Sharing Docker Compose fragments
- GitOps repositories
- Multi-environment application configuration
- Platform engineering and internal developer platforms

## Usage

### Basic Usage

Point `yamll` at your root YAML file and it pulls the graph together:

```sh
yamll build -f import.yaml
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

Importing `root.yaml` generates the combined YAML stream below:

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

### Authentication
- Pass credentials through `environment` variables and `yamll` will resolve them at runtime
- Git imports support both `ssh` and `http` URLs
- OCI imports work with registry-hosted config bundles and artifacts
- All supported authentication parameters are defined [here](https://github.com/nikhilsbhat/yamll/blob/main/pkg/yamll/dependency.go#L34)

### Dependency Graph

Visualize your dependency graph.
`yamll tree` prints it like a filesystem tree.

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

See every file affected by a change.
`yamll impact` walks the graph in reverse and shows every downstream file that depends on the target.

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

Want a fast sanity check? `yamll lint` scans the graph for duplicate keys, unresolved imports, unused imports, circular refs, invalid anchors, conflicting merges, and dependency hygiene issues.

**Example**:

```sh
yamll lint -f import.yaml
```

Output includes a severity, rule code, file, and message. Error-level issues exit non-zero; warning-level dependency advisories are printed without failing the command.

### Trace Rendered Values

Trace any rendered value back to its source.
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

### Explain Resolution

Need to know why a rendered value won?

`yamll explain` shows the final value, its final origin, and the imported or merged values that were overridden along the way.

**Example**:

```sh
yamll explain root.yaml:service.name
yamll explain -f root.yaml service.name
```

**Output**:

```text
path: service.name

final:
  origin: root.yaml:4
  value:
    api

resolution:
  1. origin: base.yaml:2
     source: merge
     action: overridden
     value:
       base-api
  2. origin: root.yaml:4
     source: direct
     action: selected
     value:
       api

merge rule: direct keys override merged values; root content is evaluated after imported anchors
```

### Reproducible Builds

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

## Why not Helm or Kustomize?

Helm and Kustomize are excellent Kubernetes tools, but they are tightly coupled to the Kubernetes ecosystem.

Yamll works with **any YAML**, whether it is Kubernetes, GitHub Actions, Docker Compose, CI/CD pipelines, application configuration, or custom formats.

| Tool      | Primary Purpose                    |
|-----------|------------------------------------|
| Yamll     | Generic YAML dependency management |
| Helm      | Kubernetes templating              |
| Kustomize | Kubernetes overlays                |
| yq        | YAML transformation                |

## Documentation

Updated documentation on all available commands and flags can be
found [here](https://github.com/nikhilsbhat/yamll/blob/main/docs/doc/yamll.md).

## Examples

Practical examples are available in the [`examples/`](examples/) directory. Each example includes a root file, shared imports, and a rendered `final.yaml`.

- [Kubernetes](examples/kubernetes/deployment.yaml)
- [GitHub Actions](examples/github-actions/ci.yaml)
- [Docker Compose](examples/docker-compose/compose.yaml)
- [Helm Values](examples/helm-values/values.yaml)
- [ArgoCD](examples/argocd/application.yaml)
- [GitOps](examples/gitops/prod.yaml)
- [GoCD](examples/gocd/pipeline.yaml)

Contributions and additional examples are always welcome.
