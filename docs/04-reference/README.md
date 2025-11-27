# Reference

Complete technical documentation of Zenith Operator API and troubleshooting guide.

## Contents

### [Function CRD - Complete Specification](function-crd.md)
Complete reference of the `Function` Custom Resource Definition.

**Topics covered:**
- API Group and Version
- All Spec fields (gitRepo, build, deploy, eventing, observability)
- All Status fields (conditions, imageDigest, url)
- Complete examples for each configuration
- Status progression during lifecycle
- Validations and constraints

**Use this document when:**
- You need to know all available fields
- You want to understand configuration options
- You are writing YAML manifests
- You need specific configuration examples

### [Operator Reference](operator-reference.md)
Internal behavior of the operator and its integrations.

**Topics covered:**
- Reconciliation loop and triggers
- Integration with Tekton (PipelineRun, ServiceAccount, image digest)
- Integration with Knative (Service, auto-scaling, URLs)
- Integration with Dapr (sidecar injection, features)
- Authentication and Secrets (Git, Registry)
- Environment variables

**Use this document when:**
- You want to understand how the operator works internally
- You need to debug reconciliation issues
- You are configuring authentication
- You want to understand integrations with other technologies

### [Troubleshooting](troubleshooting.md)
Complete guide to troubleshooting and debugging.

**Topics covered:**
- Useful diagnostic commands
- Common issues and solutions:
  - Build fails (Git auth, buildpack, registry)
  - Function does not respond (port, startup, cold start)
  - Events do not arrive (broker, filters, trigger)
  - URL not accessible (gateway, DNS)
- Logs and debugging
- Configuration validation

**Use this document when:**
- You encounter errors or problems
- You need to debug a function
- You want to validate your configuration
- You need to collect information to report a bug

## API Structure

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: example
spec:
  gitRepo: https://github.com/org/repo
  gitRevision: main
  gitAuthSecretName: github-auth
  build:
    image: registry.io/image
    registrySecretName: registry-creds
  deploy:
    dapr:
      enabled: true
      appID: example
      appPort: 8080
    env: []
    envFrom: []
  eventing:
    broker: default
    filters:
      type: event.type
  observability:
    tracing:
      enabled: true
      samplingRate: "0.1"
status:
  conditions: []
  imageDigest: registry.io/image@sha256:...
  url: http://example.default.svc.cluster.local
  observedGeneration: 1
```

## Next Steps

- **[Guides](../02-guides/)** - Apply reference in practical tutorials
- **[Concepts](../03-concepts/)** - Understand the architecture behind the API
- **[Operations](../05-operations/)** - Configure production environment
