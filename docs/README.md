# Yamll Documentation

Yamll lets you treat YAML like source code. Instead of copying the same snippets across repositories and environments, you can import reusable YAML files, render a final output, inspect the dependency graph, trace values back to their source, and lock remote imports for reproducible builds.

## What Yamll Helps With

| Problem | Yamll feature |
|---------|---------------|
| Repeated YAML across services | Local, Git, OCI, and HTTP imports |
| Unclear final rendered values | `yamll trace` and `yamll explain` |
| Risky remote dependencies | `yamll lock` and `yamll lock verify` |
| Large configuration graphs | `yamll tree` and `yamll impact` |
| Broken anchors or imports | `yamll lint` |

## Common Use Cases

- Kubernetes manifests
- Helm values
- GitHub Actions workflows
- GoCD pipelines
- Docker Compose files
- ArgoCD and GitOps configuration
- Shared application configuration

## Core Workflow

```bash
# Render a root YAML file with all imports resolved.
yamll build -f app.yaml

# See the dependency graph.
yamll tree -f app.yaml

# Find where a final value came from.
yamll trace app.yaml:metadata.labels.app

# Explain why a value won.
yamll explain app.yaml:spec.replicas

# Catch import and merge problems early.
yamll lint -f app.yaml

# Pin remote imports for reproducible builds.
yamll lock -f app.yaml
```

## Next Steps

- Start with the [getting started guide](getting-started.md).
- Learn how imports work in [imports](concepts/imports.md).
- See practical examples in the [examples hub](examples/index.md).
- Use the [command reference](reference/commands.md) when you need exact flags.
