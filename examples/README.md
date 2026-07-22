# Yamll Examples

These examples show how Yamll can reuse shared YAML across common configuration formats.

Run any example with:

```sh
yamll build -f examples/kubernetes/deployment.yaml --no-lock
yamll trace examples/kubernetes/deployment.yaml:spec.template.spec.containers.0.image --no-lock
yamll explain examples/kubernetes/deployment.yaml:metadata.labels --no-lock
```

The examples use only local imports, so they work without network access or registry credentials.

## Examples

| Example | Root file | Final output | What it shows |
|---------|-----------|--------------|---------------|
| Kubernetes | `kubernetes/deployment.yaml` | `kubernetes/final.yaml` | shared labels, pod defaults, and container defaults |
| GitHub Actions | `github-actions/ci.yaml` | `github-actions/final.yaml` | reusable workflow permissions and Go job setup |
| Docker Compose | `docker-compose/compose.yaml` | `docker-compose/final.yaml` | shared service defaults across app services |
| Helm Values | `helm-values/values.yaml` | `helm-values/final.yaml` | reusable values for services, resources, and probes |
| ArgoCD | `argocd/application.yaml` | `argocd/final.yaml` | reusable Application metadata and destination settings |
| GitOps | `gitops/prod.yaml` | `gitops/final.yaml` | reusable environment metadata and application config |
| GoCD | `gocd/pipeline.yaml` | `gocd/final.yaml` | reusable pipeline defaults, materials, and build tasks |
