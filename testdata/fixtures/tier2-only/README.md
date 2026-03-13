# Static Site

A static website with no programming languages — only HTML, CSS, and infrastructure config.

## Deployment

```bash
# Docker
docker-compose up -d

# Kubernetes
kubectl apply -f infrastructure/k8s/

# Terraform
cd infrastructure/terraform && terraform apply
```
