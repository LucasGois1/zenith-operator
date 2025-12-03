# Zenith Operator Helm Chart

A comprehensive Helm chart for deploying the Zenith Operator and all its dependencies (Tekton Pipelines, Knative Serving, Knative Eventing, Envoy Gateway, Gateway API, OpenTelemetry Operator, and optional Dapr) in a single installation.

## Overview

The Zenith Operator provides a serverless function platform on Kubernetes by orchestrating:
- **Tekton Pipelines**: Building function images from Git repositories
- **Knative Serving**: Deploying functions with auto-scaling
- **Knative Eventing**: Event-driven function invocations
- **Envoy Gateway + Gateway API**: Ingress and routing
- **OpenTelemetry Operator**: Distributed tracing and observability
- **Dapr** (optional): Service mesh capabilities

This Helm chart simplifies installation by automatically deploying all required dependencies with proper configuration and ordering.

## Prerequisites

- Kubernetes 1.33.0 or higher
- Helm 3.8.0 or higher
- kubectl configured to access your cluster
- Cluster-admin RBAC permissions
- Default StorageClass configured (for local registry in dev profile)
- Minimum cluster resources:
  - 4 CPU cores
  - 8 GB memory
  - 20 GB storage

## Supported Environments

Zenith Operator can be installed in different types of Kubernetes clusters. The main difference is LoadBalancer support:

| Environment | Values File | Reason |
|-------------|-------------|--------|
| **kind** | `values-dev.yaml` | kind does not have native LoadBalancer support (MetalLB included) |
| **Minikube** | `values-dev.yaml` | Minikube does not have native LoadBalancer support (MetalLB included) |
| **GKE** | `values.yaml` (default) | Google Cloud provides native LoadBalancer |
| **EKS** | `values.yaml` (default) | AWS provides native LoadBalancer |
| **AKS** | `values.yaml` (default) | Azure provides native LoadBalancer |
| **Bare-metal** | Depends | Can use `values-dev.yaml` with MetalLB or custom configuration |

> **Important:** MetalLB is required for Envoy Gateway to receive an external IP and for HTTP routes to work correctly. Without it, LoadBalancer Services remain in "Pending" state indefinitely.

## Quick Start

### Local Development (kind/Minikube)

```bash
# Add Helm repository
helm repo add zenith https://lucasgois1.github.io/zenith-operator
helm repo update

# Install with development profile
helm install zenith-operator zenith/zenith-operator \
  --create-namespace \
  --namespace zenith-operator-system \
  --wait \
  --timeout 15m
```

> **Important:** The `values-dev.yaml` file already includes all necessary configurations for local development:
> - `metallb.enabled: true` with IP auto-detection
> - Local Registry enabled
> - Dapr enabled
> - Insecure registries configuration for development

### Production (GKE/EKS/AKS)

```bash
# Install WITHOUT MetalLB (cloud provider provides LoadBalancer)
helm install zenith-operator zenith/zenith-operator \
  --create-namespace \
  --namespace zenith-operator-system \
  --wait \
  --timeout 15m
```

> **Important:** In production environments on managed clouds, DO NOT enable MetalLB. The cloud native load balancer is more reliable and integrated.

### Local Installation (from source)

```bash
# Clone repository
git clone https://github.com/LucasGois1/zenith-operator.git
cd zenith-operator

# For local development (kind/Minikube)
helm install zenith-operator ./charts/zenith-operator \
  --create-namespace \
  --namespace zenith-operator-system \
  --wait \
  --timeout 15m

# For production (GKE/EKS/AKS)
helm install zenith-operator ./charts/zenith-operator \
  --create-namespace \
  --namespace zenith-operator-system \
  --wait \
  --timeout 15m
```

## Installation Profiles

### Development Profile (`values-dev.yaml`)

Optimized for local development with kind/Minikube:
- ✅ All dependencies enabled (Tekton, Knative, Envoy Gateway, OpenTelemetry, Dapr)
- ✅ MetalLB enabled with IP auto-detection
- ✅ Local Registry included
- ✅ Insecure registries configuration for development
- ✅ Lower resources to run on local machines

```bash
helm install zenith-operator ./charts/zenith-operator \
  --namespace zenith-operator-system \
  --create-namespace
```

### Standard Profile (`values.yaml`)

Default configurations for production on managed clouds (GKE/EKS/AKS):
- ✅ All dependencies enabled
- ❌ MetalLB disabled (uses cloud native LoadBalancer)
- ❌ Local Registry disabled (use external registry like Docker Hub, GCR, ECR)
- ✅ Production resource limits
- ❌ Dapr disabled by default

```bash
# For GKE/EKS/AKS - DO NOT enable MetalLB
helm install zenith-operator ./charts/zenith-operator \
  --namespace zenith-operator-system \
  --create-namespace
```

### Minimal Profile (Operator Only)

Installs only the operator in a cluster that already has dependencies:

```bash
helm install zenith-operator ./charts/zenith-operator \
  --set installStack=false \
  --namespace zenith-operator-system \
  --create-namespace
```

## Configuration

### Key Configuration Options

| Parameter | Description | Default | Dev Profile |
|-----------|-------------|---------|-------------|
| `installStack` | Install full platform stack | `true` | `true` |
| `profile` | Installation profile | `standard` | `dev` |
| `tekton.enabled` | Install Tekton Pipelines | `true` | `true` |
| `tekton.version` | Tekton version | `v0.68.0` | `v0.68.0` |
| `knativeServing.enabled` | Install Knative Serving | `true` | `true` |
| `knativeServing.version` | Knative Serving version | `v0.41.2` | `v0.41.2` |
| `knativeEventing.enabled` | Install Knative Eventing | `true` | `true` |
| `knativeEventing.version` | Knative Eventing version | `v0.41.7` | `v0.41.7` |
| `envoyGateway.enabled` | Install Envoy Gateway | `true` | `true` |
| `envoyGateway.version` | Envoy Gateway version | `v1.6.0` | `v1.6.0` |
| `opentelemetry.enabled` | Install OpenTelemetry Operator | `true` | `true` |
| `opentelemetry.version` | OpenTelemetry Operator version | `v0.140.0` | `v0.140.0` |
| `dapr.enabled` | Install Dapr | `false` | `true` |
| `registry.enabled` | Install local registry | `false` | `true` |
| `operator.image.repository` | Operator image | `ghcr.io/lucasgois1/zenith-operator` | same |
| `operator.image.tag` | Operator image tag | Chart.AppVersion | `test` |

### Tekton Configuration

```yaml
tekton:
  enabled: true
  version: "v0.68.0"
  # Note: Tekton Tasks (git-clone, buildpacks-phases) are created dynamically
  # by the operator in the Function's namespace. No ClusterTasks are used.
```

### Knative Configuration

```yaml
knativeServing:
  enabled: true
  version: "v0.41.2"
  config:
    ingressClass: "gateway-api.ingress.networking.knative.dev"

knativeEventing:
  enabled: true
  version: "v0.41.7"
```

### Gateway API Configuration

```yaml
gatewayAPI:
  enabled: true
  version: "v1.3.0"
  gateway:
    name: "knative-gateway"
    namespace: "knative-serving"
    className: "envoy"
```

### Envoy Gateway Configuration

```yaml
envoyGateway:
  enabled: true
  version: "v1.6.0"
  controllerName: "gateway.envoyproxy.io/gatewayclass-controller"
```

### Dapr Configuration

```yaml
dapr:
  enabled: false  # true for dev profile
  version: "1.14.0"
  repository: "https://dapr.github.io/helm-charts/"
```

### Local Registry Configuration

```yaml
registry:
  enabled: false  # true for dev profile
  image:
    repository: registry
    tag: "2"
  storage:
    size: "10Gi"
    storageClass: "standard"
  service:
    type: ClusterIP
    port: 5000
  hostname: "registry.registry.svc.cluster.local"
```

### Operator Configuration

```yaml
operator:
  image:
    repository: ghcr.io/lucasgois1/zenith-operator
    tag: ""  # defaults to Chart.AppVersion
    pullPolicy: IfNotPresent
  
  resources:
    limits:
      cpu: 500m
      memory: 128Mi
    requests:
      cpu: 10m
      memory: 64Mi
  
  controller:
    insecureRegistries:
      - "registry.registry.svc.cluster.local:5000"
```

### Preflight Checks

```yaml
preflight:
  enabled: true
  checks:
    rbac: true
    kubernetesVersion: true
    storageClass: true
    nodeResources: true
```

## Version Compatibility Matrix

| Component | Version | Tested With |
|-----------|---------|-------------|
| Kubernetes | 1.33.0+ | 1.33.0, 1.34.0 |
| Tekton Pipelines | v0.68.0 | v0.68.0 |
| Knative Serving | v0.41.2 | v0.41.2 |
| Knative Eventing | v0.41.7 | v0.41.7 |
| Gateway API | v1.3.0 | v1.3.0 |
| net-gateway-api | knative-v1.20.0 | knative-v1.20.0 |
| Envoy Gateway | v1.6.0 | v1.6.0 |
| OpenTelemetry Operator | v0.140.0 | v0.140.0 |
| cert-manager | v1.16.2 | v1.16.2 |
| Dapr | 1.14.0 | 1.14.0 |

## Installation Order

The Helm chart uses hooks to ensure proper installation ordering:

1. **Pre-install** (weight: -5): Preflight checks
2. **Install** (weight: 0): 
   - CRDs (Helm automatic)
   - Gateway API CRDs
   - Tekton Pipelines
   - Knative Serving CRDs
   - Knative Serving Core
   - Knative Eventing CRDs
   - Knative Eventing Core
   - net-gateway-api
   - Envoy Gateway
   - cert-manager (required for OpenTelemetry Operator)
   - OpenTelemetry Operator
   - Dapr (via dependency)
   - Local Registry
   - Operator Deployment
3. **Post-install** (weight: 10-15): Knative configuration job

## Usage Examples

### Create a Simple Function

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: hello-world
  namespace: default
spec:
  git:
    url: https://github.com/your-org/hello-function
    revision: main
  image: registry.registry.svc.cluster.local:5000/hello-world:latest
  builder: paketobuildpacks/builder:base
```

### Create a Function with Dapr

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: dapr-function
  namespace: default
  annotations:
    dapr.io/enabled: "true"
spec:
  git:
    url: https://github.com/your-org/dapr-function
    revision: main
  image: registry.registry.svc.cluster.local:5000/dapr-function:latest
  builder: paketobuildpacks/builder:base
  deploy:
    dapr:
      enabled: true
      appID: dapr-function
      appPort: 8080
```

### Create an Event-Driven Function

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: event-handler
  namespace: default
spec:
  git:
    url: https://github.com/your-org/event-handler
    revision: main
  image: registry.registry.svc.cluster.local:5000/event-handler:latest
  builder: paketobuildpacks/builder:base
  eventing:
    broker: default
    filters:
      type: com.example.order.created
```

## Verification

### Check Installation Status

```bash
# Check operator
kubectl get pods -n zenith-operator-system
kubectl logs -n zenith-operator-system -l app.kubernetes.io/name=zenith-operator -f

# Check dependencies
kubectl get pods -n tekton-pipelines
kubectl get pods -n knative-serving
kubectl get pods -n knative-eventing
kubectl get pods -n envoy-gateway-system

# Check CRDs
kubectl get crds | grep -E "tekton|knative|gateway|functions"

# Check Tekton Tasks (created by operator in Function namespaces)
kubectl get tasks -A -l app.kubernetes.io/managed-by=zenith-operator
```

### Test Function Deployment

```bash
# Create a test function
kubectl apply -f - <<EOF
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: test-function
  namespace: default
spec:
  git:
    url: https://github.com/buildpacks/samples
    revision: main
  image: registry.registry.svc.cluster.local:5000/test-function:latest
  builder: paketobuildpacks/builder:base
EOF

# Watch the build
kubectl get pipelineruns -n default -w

# Check the deployment
kubectl get ksvc -n default
kubectl get functions -n default
```

## Troubleshooting

### Preflight Checks Failed

```bash
# View preflight check logs
kubectl logs -n zenith-operator-system -l app.kubernetes.io/name=zenith-preflight-checks

# Common issues:
# - Insufficient RBAC: Ensure you have cluster-admin permissions
# - Kubernetes version: Upgrade to 1.33.0+
# - No default StorageClass: Create one or specify in values
```

### Tekton Build Failures

```bash
# Check PipelineRun status
kubectl get pipelineruns -n <namespace>
kubectl describe pipelinerun <name> -n <namespace>

# Check TaskRun logs
kubectl logs -n <namespace> <pipelinerun>-fetch-source-pod --all-containers
kubectl logs -n <namespace> <pipelinerun>-build-pod --all-containers

# Common issues:
# - Git authentication: Create secret with GitHub token
# - Registry authentication: Create docker-registry secret
# - Insecure registry: Add to operator.controller.insecureRegistries
```

### Knative Service Not Ready

```bash
# Check Knative Service status
kubectl get ksvc -n <namespace>
kubectl describe ksvc <name> -n <namespace>

# Check pods
kubectl get pods -n <namespace>
kubectl logs -n <namespace> <pod-name>

# Common issues:
# - Image pull errors: Check registry credentials
# - Resource limits: Increase node resources
# - Ingress not configured: Verify Kong and Gateway API
```

### Envoy Gateway Issues

```bash
# Check Envoy Gateway status
kubectl get pods -n envoy-gateway-system
kubectl logs -n envoy-gateway-system -l control-plane=envoy-gateway

# Check Gateway
kubectl get gateway -n knative-serving
kubectl describe gateway knative-gateway -n knative-serving

# Check GatewayClass
kubectl get gatewayclass
kubectl describe gatewayclass envoy

# Common issues:
# - Gateway not ready: Wait for Envoy Gateway to be ready
# - HTTPRoutes not created: Check net-gateway-api controller logs
```

## Upgrading

### Upgrade the Chart

```bash
# Update Helm repository
helm repo update

# Upgrade with same values
helm upgrade zenith-operator zenith/zenith-operator \
  --namespace zenith-operator-system \
  --reuse-values \
  --wait \
  --timeout 15m

# Upgrade with new values
helm upgrade zenith-operator zenith/zenith-operator \
  --namespace zenith-operator-system \
  --wait \
  --timeout 15m
```

### Upgrade Strategy

The chart uses Helm hooks to ensure safe upgrades:
1. Pre-upgrade preflight checks verify cluster compatibility
2. Post-upgrade configuration job updates Knative settings
3. Operator deployment uses RollingUpdate strategy

## Uninstallation

```bash
# Uninstall the chart
helm uninstall zenith-operator --namespace zenith-operator-system

# Clean up CRDs (optional, will delete all Functions)
kubectl delete crd functions.zenith.com

# Clean up namespaces
kubectl delete namespace zenith-operator-system
kubectl delete namespace tekton-pipelines
kubectl delete namespace knative-serving
kubectl delete namespace knative-eventing
kubectl delete namespace envoy-gateway-system
kubectl delete namespace registry
kubectl delete namespace dapr-system  # if Dapr was installed
```

## Development

### Testing the Chart Locally

```bash
# Lint the chart
helm lint ./charts/zenith-operator

# Template the chart (dry-run)
helm template zenith-operator ./charts/zenith-operator \
  --namespace zenith-operator-system

# Install in Kind cluster
kind create cluster --name zenith-test
helm install zenith-operator ./charts/zenith-operator \
  --namespace zenith-operator-system \
  --create-namespace \
  --wait \
  --timeout 15m
```

### Updating Dependencies

```bash
# Update Chart.yaml dependencies
helm dependency update ./charts/zenith-operator

# This downloads:
# - Dapr chart
```

## Contributing

Contributions are welcome! Please see the main [repository](https://github.com/LucasGois1/zenith-operator) for contribution guidelines.

## License

Apache License 2.0 - See [LICENSE](../../LICENSE) for details.

## Support

- GitHub Issues: https://github.com/LucasGois1/zenith-operator/issues
- Documentation: https://github.com/LucasGois1/zenith-operator
- Tekton: https://tekton.dev
- Knative: https://knative.dev
- Envoy Gateway: https://gateway.envoyproxy.io
- Dapr: https://dapr.io
