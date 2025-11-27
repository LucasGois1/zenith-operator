# Operations

Configuration and management of Zenith Operator in production environments.

## Contents

### [Helm Chart](helm-chart.md)
Complete Helm chart documentation for installing Zenith Operator and full stack.

**Topics covered:**
- Installation via Helm
- Values configuration
- Profiles (standard vs dev)
- Stack components (Tekton, Knative, Envoy Gateway)
- Customization and troubleshooting

**Use this document when:**
- Installing the operator for the first time
- Configuring production environment
- Understanding stack components
- Customizing installation

### [Registry Configuration](registry-configuration.md)
Complete guide to configuring container registries in production.

**Topics covered:**
- Automatic detection of insecure registries
- Docker Hub (recommended for starters)
- Custom registries (Harbor, Nexus, ECR, GCR)
- In-cluster registry for production
- Authentication and secrets
- Registry troubleshooting

**Use this document when:**
- Configuring a production registry
- Having issues with image push/pull
- Using a private registry
- Configuring registry authentication

## Production Configuration

### Production Checklist

Before using Zenith Operator in production, ensure:

- [ ] **Kubernetes Cluster** configured and stable
- [ ] **Tekton Pipelines** installed and working
- [ ] **Knative Serving** installed with Gateway configured
- [ ] **Container Registry** configured with authentication
- [ ] **Git Secrets** created for private repositories
- [ ] **Monitoring** configured (Prometheus, Grafana)
- [ ] **Logging** centralized configured
- [ ] **Backup** and disaster recovery planned
- [ ] **RBAC** properly configured
- [ ] **Network Policies** applied as needed

### Best Practices

1. **Use HTTPS** for all production registries
2. **Rotate credentials** regularly (Git tokens, registry passwords)
3. **Configure resource limits** for functions
4. **Implement monitoring** and alerts
5. **Use separate namespaces** for different environments
6. **Configure backup** of Function CRs
7. **Document** your custom configurations
8. **Test** in staging before production

### Environments

We recommend maintaining separate environments:

- **Development**: Local cluster (kind/minikube) with `dev` profile
- **Staging**: Dedicated cluster with production-like configuration
- **Production**: Dedicated cluster with high availability and monitoring

## Next Steps

- **[Guides](../02-guides/)** - Learn how to create functions
- **[Reference](../04-reference/)** - Consult complete API
- **[Troubleshooting](../04-reference/troubleshooting.md)** - Resolve issues
