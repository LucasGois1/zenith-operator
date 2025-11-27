# Concepts

Understand the architecture and fundamental concepts of Zenith Operator.

## Contents

### [Architecture](architecture.md)
Complete architecture documentation of Zenith Operator with detailed diagrams.

**Topics covered:**
- High-level overview
- Function CRD structure
- Operator reconciliation flow
- Complete developer experience
- Integration with Tekton Pipelines
- Integration with Knative Serving
- Event-driven architecture
- Key features and functionality

**Diagrams included:**
- High-level architecture
- Custom Resource structure
- Reconciliation flow
- Tekton build pipeline
- Knative Service deployment
- Event routing

### [Function Lifecycle](function-lifecycle.md)
*(In development)* Details about the complete lifecycle of a function, from creation to removal.

## How the Operator Works

Zenith Operator abstracts the complexity of multiple cloud-native technologies:

1. **Tekton Pipelines** - Builds container images from source code
2. **Knative Serving** - Deploys and manages function auto-scaling
3. **Knative Eventing** - Routes events to event-driven functions
4. **Dapr** (optional) - Provides service mesh, pub/sub, and state management

All this is controlled through a single `Function` Custom Resource, making serverless development simple and declarative.

## Next Steps

After understanding the concepts:

- **[Reference](../04-reference/)** - Consult the complete API specification
- **[Guides](../02-guides/)** - Apply concepts in practical tutorials
- **[Operations](../05-operations/)** - Configure production environment
