# Zenith Operator - Documentation

Welcome to the Zenith Operator documentation! This is a serverless platform for Kubernetes that simplifies function development and deployment through a single Custom Resource.

## 🚀 What is Zenith Operator?

Zenith Operator is a Kubernetes operator that abstracts the complexity of integrating multiple cloud-native technologies (Tekton Pipelines, Knative Serving, Knative Eventing, and Dapr) into a simple and declarative experience.

With Zenith Operator, you can:

- **Build** container images automatically from source code (no Dockerfile required)
- **Deploy** serverless functions with auto-scaling and scale-to-zero
- **Connect** functions to events for asynchronous processing
- **Communicate** between functions using HTTP or service mesh
- **Trace** distributed requests with OpenTelemetry

All this through a single `Function` Custom Resource.

## 📖 Documentation Navigation

### [01. Introduction](01-introduction/)

Start here if you are new to Zenith Operator.

- **[Overview](01-introduction/overview.md)** - Understand what the operator is and its main features
- **[Installation](01-introduction/installation.md)** - Install the operator on your Kubernetes cluster
- **[Quick Start](01-introduction/quick-start.md)** - Create your first function in 5 minutes

### [02. Guides](02-guides/)

Practical tutorials for creating different types of functions.

- **[HTTP Functions](02-guides/http-functions.md)** - REST APIs, webhooks, and synchronous microservices
- **[Event Functions](02-guides/event-functions.md)** - Asynchronous event-driven processing
- **[Function Communication](02-guides/function-communication.md)** - Distributed microservices architectures
- **[Git Authentication](02-guides/git-authentication.md)** - Configure access to private repositories
- **[Observability](02-guides/observability.md)** - Distributed tracing with OpenTelemetry

### [03. Concepts](03-concepts/)

Understand the architecture and fundamental concepts.

- **[Architecture](03-concepts/architecture.md)** - Diagrams and explanations of the complete architecture
- **[Function Lifecycle](03-concepts/function-lifecycle.md)** - How functions are created, updated, and removed

### [04. Reference](04-reference/)

Complete technical API documentation.

- **[Function CRD](04-reference/function-crd.md)** - Complete specification of all fields
- **[Operator Reference](04-reference/operator-reference.md)** - Internal behavior and integrations
- **[Troubleshooting](04-reference/troubleshooting.md)** - Common issues troubleshooting

### [05. Operations](05-operations/)

Configuration and management in production.

- **[Helm Chart](05-operations/helm-chart.md)** - Helm installation and stack configuration
- **[Registry Configuration](05-operations/registry-configuration.md)** - Container registry setup

## 🎯 Common Use Cases

### Synchronous REST API

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

### Event Processing

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
```

### Microservices with Service Mesh

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: payment-service
spec:
  gitRepo: https://github.com/myorg/payment-service
  gitRevision: main
  build:
    image: registry.example.com/payment-service:latest
  deploy:
    dapr:
      enabled: true
      appID: payment-service
      appPort: 8080
```

## 🚦 Quick Start

1. **Install the operator** following the [installation guide](01-introduction/installation.md)

2. **Create your first function** with the [quick start tutorial](01-introduction/quick-start.md)

3. **Explore the guides** to learn advanced features:
   - [HTTP Functions](02-guides/http-functions.md)
   - [Event Functions](02-guides/event-functions.md)
   - [Function Communication](02-guides/function-communication.md)

## 🔍 Finding What You Need

### I'm just starting
→ Start with [Introduction](01-introduction/) and follow the [Quick Start](01-introduction/quick-start.md)

### I want to create an HTTP function
→ See the [HTTP Functions](02-guides/http-functions.md) guide

### I want to process events
→ See the [Event Functions](02-guides/event-functions.md) guide

### I need to configure Git authentication
→ See the [Git Authentication](02-guides/git-authentication.md) guide

### I'm having problems
→ Consult [Troubleshooting](04-reference/troubleshooting.md)

### I need the complete API reference
→ See [Function CRD](04-reference/function-crd.md)

### I want to understand how it works internally
→ Read about [Architecture](03-concepts/architecture.md) and [Operator Reference](04-reference/operator-reference.md)

## 🤝 Contributing

Contributions are welcome! Visit the [GitHub repository](https://github.com/LucasGois1/zenith-operator) to:

- Report bugs and issues
- Suggest new features
- Contribute with code
- Improve documentation

## 📄 License

This project is licensed under the Apache License 2.0.

## 🔗 Useful Links

- [GitHub Repository](https://github.com/LucasGois1/zenith-operator)
- [Function Examples](https://github.com/LucasGois1/zenith-test-functions)
- [Issues and Support](https://github.com/LucasGois1/zenith-operator/issues)
