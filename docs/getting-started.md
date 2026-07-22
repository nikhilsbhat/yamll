# Getting Started

This guide shows the smallest useful Yamll setup: one root YAML file and one shared library file.

## Install Yamll

Download a release from GitHub, install with Homebrew, use the container image, or build from source.

```bash
brew tap nikhilsbhat/stable git@github.com:nikhilsbhat/homebrew-stable.git
brew install nikhilsbhat/stable/yamll
```

```bash
docker pull ghcr.io/nikhilsbhat/yamll:latest
```

## Create a Shared YAML Library

Create `base.yaml`:

```yaml
defaults: &defaults
  labels:
    app.kubernetes.io/managed-by: yamll
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
```

## Import It From a Root File

Create `service.yaml`:

```yaml
##++base.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: payments-api
  <<: *defaults
spec:
  replicas: 3
```

The `##++base.yaml` comment tells Yamll to load `base.yaml` before rendering the root file. The root file can then use anchors and aliases from the imported library.

## Render the Final YAML

```bash
yamll build -f service.yaml
```

Write the rendered output to a file:

```bash
yamll build -f service.yaml --to-file final.yaml
```

## Inspect the Graph

```bash
yamll tree -f service.yaml
```

Use structured formats when another tool needs to consume the graph:

```bash
yamll tree -f service.yaml --output=json
yamll tree -f service.yaml --output=dot
yamll tree -f service.yaml --output=mermaid
```

## Debug a Value

Trace where a final value came from:

```bash
yamll trace service.yaml:metadata.labels.app.kubernetes.io/managed-by
```

Explain why a value was selected:

```bash
yamll explain service.yaml:metadata.labels
```

## Keep Remote Imports Reproducible

When your imports include Git, HTTP, or OCI sources, generate a lock file:

```bash
yamll lock -f service.yaml
```

Then verify the lock in CI:

```bash
yamll lock verify -f service.yaml
yamll build -f service.yaml
```
