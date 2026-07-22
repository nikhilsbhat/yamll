# Usage Patterns

These patterns work well when a repository grows beyond a few YAML files.

## Shared Base, Thin Roots

Keep common labels, probes, resources, permissions, and defaults in shared files. Keep environment-specific files small.

```yaml
##++common/kubernetes.yaml
##++common/resources.yaml

metadata:
  name: payments-api
  <<: *app_metadata
spec:
  replicas: 3
```

## Environment Layers

Use one root per environment and import shared fragments.

```yaml
##++common/base.yaml
##++environments/prod/*.yaml

environment:
  name: production
```

Then render each environment separately:

```bash
yamll build -f environments/dev.yaml --to-file dist/dev.yaml
yamll build -f environments/prod.yaml --to-file dist/prod.yaml
```

## Shared Platform Repository

Keep reusable platform YAML in a central Git repository and import versioned files from application repositories.

```yaml
##++github.com/org/platform-config//kubernetes/service.yaml@v1.2.0
```

Generate and commit a lock file so builds do not drift:

```bash
yamll lock -f service.yaml
```

## CI Pipeline

A typical CI job validates the graph, verifies the lock, and renders the final YAML.

```bash
yamll lint -f service.yaml
yamll lock verify -f service.yaml
yamll build -f service.yaml --to-file final.yaml
```

## Release Artifacts

Use `--to-file` when the rendered YAML should be published, deployed, or reviewed.

```bash
yamll build -f examples/kubernetes/deployment.yaml --no-lock --to-file final.yaml
```

The examples in this repository include checked-in `final.yaml` files so readers can compare the source imports with the rendered result.
