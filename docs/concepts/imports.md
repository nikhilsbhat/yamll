# Imports

Yamll imports are declared as comments that start with `##++`. The imported file is loaded before the root content, which lets the root file use anchors, aliases, and reusable maps from the imported YAML.

```yaml
##++common/base.yaml

service:
  <<: *service_defaults
  name: payments-api
```

## Local Imports

Use local imports when shared YAML lives in the same repository.

```yaml
##++common/base.yaml
##++common/resources.yaml
```

Local imports are resolved relative to the root file.

## Pattern Imports

Use glob patterns when a directory contains several YAML fragments.

```yaml
##++environments/prod/*.yaml
```

Pattern imports are useful for large repositories because a root file can depend on a group of files without listing every file manually. Yamll sorts matched files deterministically so repeated builds stay stable.

## Git Imports

Git imports let you reuse YAML from another repository.

```yaml
##++git+https://github.com/org/platform-config@v1.2.0?path=base/service.yaml
```

Yamll also supports shorthand syntax:

```yaml
##++github.com/org/platform-config//base/service.yaml@v1.2.0
```

For private repositories, pass credentials through environment variables and reference them in the import metadata.

```yaml
##++git+https://github.com/org/private-config@main?path=base.yaml;{"user_name":"${GIT_USERNAME}","password":"${GITHUB_TOKEN}"}
```

## HTTP Imports

Use HTTP imports for configuration served by an internal endpoint.

```yaml
##++https://config.example.com/platform/base.yaml
```

## OCI Imports

OCI imports let you distribute YAML bundles through registries used by Helm, GitOps, and container platforms.

```yaml
##++oci://ghcr.io/company/platform-config:v1
```

## Merge Rules

Yamll renders the dependency graph first and then applies the root file. Direct keys in the root file override values imported through YAML merges.

```yaml
##++base.yaml

service:
  <<: *service_defaults
  port: 8080
```

If `service_defaults` also defines `port`, the root value wins.
