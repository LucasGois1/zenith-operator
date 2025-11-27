# Zenith Operator Installation Guide

This guide explains how to install the Zenith Operator using Helm.

## Prerequisites

- Kubernetes cluster (v1.33.0 or later required)
- Helm 3.x installed
- `kubectl` configured to access your cluster

## Installation

### Step 1: Add the Helm Repository

Add the Zenith Operator Helm repository:

```bash
helm repo add zenith-operator https://lucasgois1.github.io/zenith-operator
helm repo update
```

### Step 2: Install the Operator

**Para Desenvolvimento Local (kind/Minikube):**

```bash
# Baixar o values-dev.yaml (já inclui MetalLB, registry local, Dapr, etc.)
curl -O https://raw.githubusercontent.com/LucasGois1/zenith-operator/main/charts/zenith-operator/values-dev.yaml

# Instalar com o profile de desenvolvimento
helm install zenith-operator zenith-operator/zenith-operator \
  -f values-dev.yaml
```

> **Importante:** O `values-dev.yaml` já vem configurado com MetalLB habilitado, registry local, e outras configurações otimizadas para desenvolvimento. Isso é obrigatório em clusters locais (kind/Minikube) para que o Envoy Gateway receba um IP externo.

**Para Produção (GKE/EKS/AKS):**

```bash
helm install zenith-operator zenith-operator/zenith-operator
```

> **Nota:** Em clouds gerenciadas, NÃO habilite o MetalLB. O load balancer nativo da cloud é usado automaticamente.

This will install the operator along with all required dependencies:
- Tekton Pipelines (for building functions)
- Knative Serving (for serverless deployments)
- Knative Eventing (for event-driven architectures)
- Envoy Gateway (for routing)
- Gateway API CRDs
- MetalLB (apenas se habilitado, para clusters locais)

### Step 3: Verify Installation

Check that all components are running:

```bash
# Check operator deployment
kubectl get deployment -n zenith-operator-system

# Check Tekton installation
kubectl get pods -n tekton-pipelines

# Check Knative Serving
kubectl get pods -n knative-serving

# Check Knative Eventing
kubectl get pods -n knative-eventing

# Check Envoy Gateway
kubectl get pods -n envoy-gateway-system
```

## Configuration Options

### Custom Installation

You can customize the installation by providing your own values file:

```bash
helm install zenith-operator zenith-operator/zenith-operator -f custom-values.yaml
```

### Common Configuration Options

#### Disable Optional Components

If you don't need certain components, you can disable them:

```yaml
# custom-values.yaml
dapr:
  enabled: false  # Disable Dapr if not needed

registry:
  enabled: true  # Enable local registry for development
```

#### Configure Operator Image

Use a specific operator image version:

```yaml
operator:
  image:
    repository: ghcr.io/lucasgois1/zenith-operator
    tag: "v0.1.0"
    pullPolicy: IfNotPresent
```

#### Configure Resource Limits

Adjust resource limits for the operator:

```yaml
operator:
  resources:
    limits:
      cpu: 1000m
      memory: 256Mi
    requests:
      cpu: 100m
      memory: 128Mi
```

### Installation Profiles

The chart supports different installation profiles:

```bash
# Standard profile (default) - includes all components
helm install zenith-operator zenith-operator/zenith-operator

# Minimal profile - only essential components
helm install zenith-operator zenith-operator/zenith-operator --set profile=minimal

# Development profile - includes local registry
helm install zenith-operator zenith-operator/zenith-operator --set profile=dev
```

## Upgrading

To upgrade to a newer version:

```bash
helm repo update
helm upgrade zenith-operator zenith-operator/zenith-operator
```

## Uninstallation

To uninstall the operator and all components:

```bash
helm uninstall zenith-operator
```

**Note:** This will remove the operator but may leave some CRDs and custom resources. To completely clean up:

```bash
# Remove Function CRDs
kubectl delete crd functions.functions.zenith.com

# Remove any remaining Function resources
kubectl delete functions --all --all-namespaces
```

## Troubleshooting

### Check Operator Logs

```bash
kubectl logs -n zenith-operator-system deployment/zenith-operator-controller-manager -f
```

### Verify CRDs are Installed

```bash
kubectl get crd | grep zenith
```

### Check Helm Release Status

```bash
helm status zenith-operator
helm get values zenith-operator
```

## Next Steps

After installation, you can:

1. Create your first Function - see [examples](./config/samples/)
2. Configure Git authentication - see [Git Authentication Guide](./docs/GIT_AUTHENTICATION.md)
3. Set up registry credentials for private registries
4. Explore event-driven architectures with Knative Eventing

## Support

For issues, questions, or contributions:
- GitHub Issues: https://github.com/LucasGois1/zenith-operator/issues
- Documentation: https://github.com/LucasGois1/zenith-operator

## Version Compatibility

| Operator Version | Kubernetes | Tekton | Knative Serving | Knative Eventing |
|-----------------|------------|--------|-----------------|------------------|
| 0.1.x           | 1.30+      | 0.68.0 | 0.41.2          | 0.41.7           |
