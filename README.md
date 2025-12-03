# Zenith Operator

[![codecov](https://codecov.io/github/LucasGois1/zenith-operator/branch/main/graph/badge.svg?token=2QNLMH3D7H)](https://codecov.io/github/LucasGois1/zenith-operator)
[![Lint](https://github.com/LucasGois1/zenith-operator/actions/workflows/lint.yml/badge.svg)](https://github.com/LucasGois1/zenith-operator/actions/workflows/lint.yml)
[![Tests](https://github.com/LucasGois1/zenith-operator/actions/workflows/test.yml/badge.svg)](https://github.com/LucasGois1/zenith-operator/actions/workflows/test.yml)
[![E2E Tests](https://github.com/LucasGois1/zenith-operator/actions/workflows/test-e2e.yml/badge.svg)](https://github.com/LucasGois1/zenith-operator/actions/workflows/test-e2e.yml)

Zenith Operator is a Kubernetes operator that provides a serverless platform for functions, orchestrating builds (Tekton Pipelines), deployments (Knative Serving), and event-driven invocations (Knative Eventing) through a single `Function` Custom Resource.

## 🚀 Quick Start

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: hello-function
spec:
  gitRepo: https://github.com/myorg/hello-function
  gitRevision: main
  build:
    image: registry.example.com/hello-function:latest
  deploy: {}
```

## 📖 Documentation

**[Access full documentation →](docs/)**

- **[Introduction](docs/01-introduction/)** - Overview, installation and quick start
- **[Guides](docs/02-guides/)** - Practical tutorials for creating functions
- **[Concepts](docs/03-concepts/)** - Architecture and fundamental concepts
- **[Reference](docs/04-reference/)** - Complete API specification
- **[Operations](docs/05-operations/)** - Configuration and management

## ✨ Key Features

- **Automatic Build**: Clones Git repositories and builds images using Tekton Pipelines and Buildpacks
- **Serverless Deployment**: Automatic deployment as Knative Services with scale-to-zero
- **Event-Driven**: Event subscription via Knative Eventing with filters
- **Service Mesh**: Optional integration with Dapr for service discovery and pub/sub
- **Distributed Tracing**: Automatic tracing via OpenTelemetry
- **Immutable Images**: Image digest tracking for reproducibility

## 🛠️ Installation

### Via Helm

**Local Development (kind/Minikube):**
```bash
helm repo add zenith https://lucasgois1.github.io/zenith-operator

# Install with development profile (includes MetalLB, local registry, etc.)
helm install zenith-operator zenith/zenith-operator \
  --namespace zenith-operator-system \
  --create-namespace
```

**Production (GKE/EKS/AKS):**
```bash
helm repo add zenith https://lucasgois1.github.io/zenith-operator
helm install zenith-operator zenith/zenith-operator \
  --namespace zenith-operator-system \
  --create-namespace
```

> **Note:** MetalLB is only required on local clusters (kind/Minikube) that do not have native LoadBalancer support. On managed clouds (GKE, EKS, AKS), the cloud load balancer is used automatically.

### Via Kustomize

```bash
make install  # Install CRDs
make deploy IMG=ghcr.io/lucasgois1/zenith-operator:latest
```

**[Complete installation guide →](docs/01-introduction/installation.md)**

## 🎯 Use Cases

### Synchronous HTTP Functions
REST APIs, webhooks and microservices that respond to HTTP requests.

**[See guide →](docs/02-guides/http-functions.md)**

### Asynchronous Functions with Events
Event processing, notifications and event-driven workflows.

**[See guide →](docs/02-guides/event-functions.md)**

### Function Communication
Microservices architectures with multiple communicating functions.

**[See guide →](docs/02-guides/function-communication.md)**

## 🧪 Development

```bash
# Complete environment setup
make dev-up

# Fast rebuild and redeploy
make dev-redeploy

# Run tests
make test
make test-chainsaw
```

## 📄 License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## 🤝 Contributing

Contributions are welcome! Open issues and pull requests on GitHub.

## 🔗 Links

- [Documentation](docs/)
- [Examples](config/samples/)
- [Issues](https://github.com/LucasGois1/zenith-operator/issues)
