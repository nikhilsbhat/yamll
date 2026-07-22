# Examples

The repository includes runnable examples for common YAML-heavy tools. Each example has:

- A root YAML file
- One shared import file
- A rendered `final.yaml`

The examples use only local imports, so they work without network access or registry credentials.

## Example Index

| Example | Root file | Final output | What it shows |
|---------|-----------|--------------|---------------|
| Kubernetes | [deployment.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/kubernetes/deployment.yaml) | [final.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/kubernetes/final.yaml) | shared labels, pod defaults, and container defaults |
| GitHub Actions | [ci.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/github-actions/ci.yaml) | [final.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/github-actions/final.yaml) | reusable workflow permissions and Go job setup |
| GoCD | [pipeline.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/gocd/pipeline.yaml) | [final.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/gocd/final.yaml) | reusable pipeline defaults, materials, and build tasks |
| Docker Compose | [compose.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/docker-compose/compose.yaml) | [final.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/docker-compose/final.yaml) | shared service defaults across app services |
| Helm Values | [values.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/helm-values/values.yaml) | [final.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/helm-values/final.yaml) | reusable values for services, resources, and probes |
| ArgoCD | [application.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/argocd/application.yaml) | [final.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/argocd/final.yaml) | reusable Application metadata and destination settings |
| GitOps | [prod.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/gitops/prod.yaml) | [final.yaml](https://github.com/nikhilsbhat/yamll/blob/main/examples/gitops/final.yaml) | reusable environment metadata and application config |

## Run an Example Locally

```bash
yamll build -f examples/kubernetes/deployment.yaml --no-lock
yamll trace examples/kubernetes/deployment.yaml:spec.template.spec.containers.0.image --no-lock
yamll explain examples/kubernetes/deployment.yaml:metadata.labels --no-lock
```
