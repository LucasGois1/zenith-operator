# Zenith Operator

[![Lint](https://github.com/LucasGois1/zenith-operator/actions/workflows/lint.yml/badge.svg)](https://github.com/LucasGois1/zenith-operator/actions/workflows/lint.yml)
[![Tests](https://github.com/LucasGois1/zenith-operator/actions/workflows/test.yml/badge.svg)](https://github.com/LucasGois1/zenith-operator/actions/workflows/test.yml)
[![E2E Tests](https://github.com/LucasGois1/zenith-operator/actions/workflows/test-e2e.yml/badge.svg)](https://github.com/LucasGois1/zenith-operator/actions/workflows/test-e2e.yml)

Zenith Operator is a Kubernetes operator that provides a serverless platform for functions, orchestrating builds (Tekton Pipelines), deployments (Knative Serving), and event-driven invocations (Knative Eventing) through a single `Function` Custom Resource.

## 🚀 Overview

Zenith Operator abstracts the complexity of integrating Tekton, Knative, and Dapr, allowing developers to define serverless functions declaratively using just one Custom Resource.

### Key Features

- **Automatic Build**: Clones Git repositories and builds container images using Tekton Pipelines and Buildpacks
- **Serverless Deployment**: Automatic deployment as Knative Services with scale-to-zero
- **Event-Driven**: Event subscription via Knative Eventing with attribute-based filters
- **Service Mesh**: Optional integration with Dapr for service discovery, pub/sub, and state management
- **Function Communication**: Native support for HTTP communication between functions
- **Distributed Tracing**: Automatic distributed tracing via OpenTelemetry to visualize request flows
- **Immutable Images**: Image digest tracking for reproducibility and security

## 📚 Documentation

### User Guides

- **[Creating Synchronous HTTP Functions](../02-guides/http-functions.md)** - How to create functions that respond to HTTP requests
- **[Creating Asynchronous Event Functions](../02-guides/event-functions.md)** - How to create functions that process asynchronous events
- **[Function Communication](../02-guides/function-communication.md)** - How to implement communication between multiple functions
- **[Observability and Distributed Tracing](../02-guides/observability.md)** - How to use OpenTelemetry to trace requests between functions

### Technical Reference

- **[Function CRD Specification](../04-reference/function-crd.md)** - Complete documentation of all Custom Resource fields
- **[Operator Reference](../04-reference/operator-reference.md)** - Internal behavior and operator integrations
- **[Git Authentication Configuration](../02-guides/git-authentication.md)** - How to configure authentication for private Git repositories
- **[Registry Configuration](../05-operations/registry-configuration.md)** - How to configure container registries

## 🎯 Use Cases

### 1. Synchronous HTTP Functions

Functions that respond to synchronous HTTP requests, ideal for REST APIs, webhooks, and microservices.

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: hello-api
spec:
  gitRepo: https://github.com/myorg/hello-function
  gitRevision: main
  build:
    image: registry.example.com/hello-api:latest
  deploy: {}
```

### 2. Asynchronous Functions with Events

Functions that process events asynchronously, ideal for data processing, notifications, and event-driven workflows.

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: order-processor
spec:
  gitRepo: https://github.com/myorg/order-processor
  gitRevision: main
  build:
    image: registry.example.com/order-processor:latest
  deploy: {}
  eventing:
    broker: default
    filters:
      type: com.example.order.created
      source: payment-service
```

### 3. Function Communication

Multiple functions communicating via HTTP, ideal for microservices architectures and distributed systems.

```yaml
# transaction-processor calls balance-manager which calls audit-logger
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: transaction-processor
spec:
  gitRepo: https://github.com/myorg/transaction-processor
  gitRevision: main
  build:
    image: registry.example.com/transaction-processor:latest
  deploy:
    env:
      - name: BALANCE_MANAGER_URL
        value: http://balance-manager.default.svc.cluster.local
```

## 🛠️ Installation

### Prerequisites

- Kubernetes 1.33.0+
- Tekton Pipelines v0.50+
- Knative Serving v1.20+
- Knative Eventing v1.20+ (optional, for event-driven functions)
- Envoy Gateway v1.6+ (for ingress)

### Installation via Helm

**For Local Development (kind/Minikube):**
```bash
# Add Helm repository
helm repo add zenith https://lucasgois1.github.io/zenith-operator

# Download values-dev.yaml (includes MetalLB, local registry, Dapr, etc.)
curl -O https://raw.githubusercontent.com/LucasGois1/zenith-operator/main/charts/zenith-operator/values-dev.yaml

# Install operator with development profile
helm install zenith-operator zenith/zenith-operator \
  -f values-dev.yaml \
  --namespace zenith-operator-system \
  --create-namespace
```

**For Production (GKE/EKS/AKS):**
```bash
# Add Helm repository
helm repo add zenith https://lucasgois1.github.io/zenith-operator

# Install operator (without MetalLB - uses cloud native LoadBalancer)
helm install zenith-operator zenith/zenith-operator \
  --namespace zenith-operator-system \
  --create-namespace
```

> **Note:** MetalLB is only required on local clusters that do not have native LoadBalancer support.

### Installation via Kustomize

```bash
# Install CRDs
make install

# Deploy operator
make deploy IMG=ghcr.io/lucasgois1/zenith-operator:latest
```

## 🚦 Quick Start

1. **Create a Secret for Git authentication** (if using private repository):

```bash
kubectl create secret generic github-auth \
  --from-literal=username=myuser \
  --from-literal=password=mytoken \
  --type=kubernetes.io/basic-auth

kubectl annotate secret github-auth \
  tekton.dev/git-0=https://github.com
```

2. **Create your first function**:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: my-first-function
spec:
  gitRepo: https://github.com/LucasGois1/zenith-test-functions
  gitRevision: main
  gitAuthSecretName: github-auth
  build:
    image: registry.example.com/my-first-function:latest
  deploy: {}
EOF
```

3. **Check status**:

```bash
kubectl get functions
kubectl describe function my-first-function
```

4. **Access the function**:

```bash
# Get function URL
FUNCTION_URL=$(kubectl get function my-first-function -o jsonpath='{.status.url}')
echo "Function URL: $FUNCTION_URL"

# Make a request
curl $FUNCTION_URL
```

## 🧪 Development

### Run tests locally

```bash
# Unit tests
make test

# E2E tests
make test-e2e

# Specific Chainsaw tests
make test-chainsaw-basic        # Basic function test
make test-chainsaw-eventing     # Eventing test
make test-chainsaw-integration  # Function integration test
```

### Local development

```bash
# Development environment setup
make dev-up

# Fast rebuild and redeploy
make dev-redeploy

# Clean environment
make dev-down
```

## 📖 Examples

See [config/samples/](config/samples/) directory for complete Function examples.

## 🤝 Contributing

Contributions are welcome! Please open issues and pull requests on GitHub.

## 📄 License

This project is licensed under the Apache License 2.0 - see [LICENSE](LICENSE) file for details.

## 🔗 Useful Links

- [Full Documentation](../README.md)
- [Quick Start](quick-start.md)
- [Examples](https://github.com/LucasGois1/zenith-operator/tree/main/config/samples)
- [Chainsaw Tests](https://github.com/LucasGois1/zenith-operator/tree/main/test/chainsaw)
- [Issues](https://github.com/LucasGois1/zenith-operator/issues)
