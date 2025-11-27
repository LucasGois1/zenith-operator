# Operator Reference - Behavior and Integrations

This documentation describes the behavior of Zenith Operator and its integrations with Tekton, Knative, and Dapr.

## Overview

Zenith Operator is a Kubernetes operator that manages the complete lifecycle of serverless functions through the `Function` Custom Resource.

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Zenith Operator                          │
│                                                                 │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐      │
│  │   Function   │──▶│ PipelineRun  │──▶│   Service    │      │
│  │  Controller  │   │   (Tekton)   │   │  (Knative)   │      │
│  └──────────────┘   └──────────────┘   └──────────────┘      │
│         │                                       │              │
│         │                                       │              │
│         └───────────────────┬───────────────────┘              │
│                             │                                  │
│                      ┌──────────────┐                         │
│                      │   Trigger    │                         │
│                      │  (Eventing)  │                         │
│                      └──────────────┘                         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Reconciliation Flow

1. **User creates Function CR** → Operator detects new resource
2. **Build Phase** → Operator creates Tekton PipelineRun
3. **PipelineRun executes** → Clones Git, builds image, pushes to registry
4. **Build completes** → Operator extracts image digest
5. **Deploy Phase** → Operator creates Knative Service
6. **Eventing Phase** (optional) → Operator creates Knative Trigger
7. **Status updated** → Function.status reflects current state

## Operator Behavior

### Reconciliation Loop

The operator continuously reconciles Functions:

1. **Watch**: Monitors changes in Function CRs
2. **Reconcile**: Processes each Function
3. **Update Status**: Updates status based on current state
4. **Requeue**: Schedules next reconciliation if needed

### Reconciliation Triggers

The operator reconciles when:
- Function CR is created
- Function CR is updated (spec changed)
- PipelineRun completes
- Knative Service changes
- Trigger changes
- Periodic reconciliation (every 10 minutes)

### Idempotency

The operator is idempotent:
- Multiple reconciliations produce the same result
- Existing resources are not recreated
- Updates are applied only when necessary

### Garbage Collection

The operator uses OwnerReferences for garbage collection:
- PipelineRuns are owned by Function
- Knative Services are owned by Function
- Triggers are owned by Function
- ServiceAccounts are owned by Function

When a Function is deleted, all owned resources are automatically deleted.

## Tekton Integration

### PipelineRun Creation

The operator creates a PipelineRun for each Function:

**Name**: `<function-name>-<timestamp>`

**Tasks**:
1. **git-clone**: Clones Git repository
2. **buildpacks-phases**: Builds image using Cloud Native Buildpacks

**Parameters**:
- `git-url`: Git repository URL
- `git-revision`: Git revision
- `image`: Target image name

**Workspaces**:
- `source`: Workspace for source code
- `cache`: Workspace for build cache

### ServiceAccount Management

The operator creates a dedicated ServiceAccount for each Function:

**Name**: `<function-name>-sa`

**Purpose**:
- Git Authentication (via secrets)
- Registry Authentication (via imagePullSecrets)

**Secrets Binding**:
- Git auth secret → `serviceAccount.secrets`
- Registry secret → `serviceAccount.imagePullSecrets`

### Image Digest Extraction

After successful build, the operator:
1. Waits for PipelineRun to complete
2. Extracts image digest from PipelineRun status
3. Updates Function.status.imageDigest

## Knative Integration

### Service Creation

The operator creates a Knative Service for each Function:

**Name**: `<function-name>`

**Spec**:
```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: my-function
  ownerReferences:
    - apiVersion: functions.zenith.com/v1alpha1
      kind: Function
      name: my-function
      uid: <function-uid>
      controller: true
      blockOwnerDeletion: true
spec:
  template:
    metadata:
      annotations:
        # Dapr annotations (if enabled)
        dapr.io/enabled: "true"
        dapr.io/app-id: "my-function"
        dapr.io/app-port: "8080"
    spec:
      containers:
        - image: registry.example.com/my-function@sha256:abc123...
          env:
            - name: DATABASE_URL
              value: postgres://db.example.com/mydb
      imagePullSecrets:
        - name: registry-credentials
```

### Auto-scaling

Knative Services auto-scale based on traffic:
- **Scale-to-zero**: Pods are terminated when no traffic
- **Scale-from-zero**: Pods are created when traffic arrives
- **Horizontal scaling**: Multiple pods for high traffic

### URL Exposure

Knative exposes URLs:
- **Internal**: `http://<service-name>.<namespace>.svc.cluster.local`
- **External**: `http://<service-name>.<namespace>.<domain>`

### Trigger Creation

If `spec.eventing` is configured, the operator creates a Trigger:

**Name**: `<function-name>-trigger`

**Spec**:
```yaml
apiVersion: eventing.knative.dev/v1
kind: Trigger
metadata:
  name: my-function-trigger
  ownerReferences:
    - apiVersion: functions.zenith.com/v1alpha1
      kind: Function
      name: my-function
spec:
  broker: default
  filter:
    attributes:
      type: com.example.order.created
      source: payment-service
  subscriber:
    ref:
      apiVersion: serving.knative.dev/v1
      kind: Service
      name: my-function
```

## Dapr Integration

### Sidecar Injection

When `spec.deploy.dapr.enabled=true`, the operator adds annotations to Knative Service:

```yaml
annotations:
  dapr.io/enabled: "true"
  dapr.io/app-id: "<appID>"
  dapr.io/app-port: "<appPort>"
```

### Dapr Features

With Dapr enabled, functions can use:

#### Service Invocation

```go
import "github.com/dapr/go-sdk/client"

daprClient, _ := client.NewClient()
resp, _ := daprClient.InvokeMethod(ctx, "other-function", "endpoint", "post")
```

#### Pub/Sub

```go
// Publish
daprClient.PublishEvent(ctx, "pubsub", "topic", data)

// Subscribe (via annotation)
// dapr.io/subscribe: '[{"pubsubname":"pubsub","topic":"topic","route":"/events"}]'
```

#### State Management

```go
// Save state
daprClient.SaveState(ctx, "statestore", "key", []byte("value"), nil)

// Get state
item, _ := daprClient.GetState(ctx, "statestore", "key", nil)
```

#### Secrets

```go
// Get secret
secret, _ := daprClient.GetSecret(ctx, "secretstore", "key", nil)
```

## Authentication and Secrets

### Git Authentication

#### HTTPS Authentication

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-auth
  annotations:
    tekton.dev/git-0: https://github.com
type: kubernetes.io/basic-auth
stringData:
  username: myusername
  password: ghp_mytoken  # GitHub Personal Access Token
```

#### SSH Authentication

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: github-ssh
  annotations:
    tekton.dev/git-0: github.com
type: kubernetes.io/ssh-auth
stringData:
  ssh-privatekey: |
    -----BEGIN OPENSSH PRIVATE KEY-----
    ...
    -----END OPENSSH PRIVATE KEY-----
```

### Registry Authentication

```bash
# Create secret
kubectl create secret docker-registry registry-credentials \
  --docker-server=registry.example.com \
  --docker-username=myuser \
  --docker-password=mypassword \
  --docker-email=myemail@example.com
```

## Environment Variables

### Variables Injected by Operator

The operator automatically injects:

- `PORT`: Port where application must listen (default: `8080`)
- `K_SERVICE`: Knative Service Name
- `K_CONFIGURATION`: Configuration Name
- `K_REVISION`: Revision Name

### Custom Variables

Add via `spec.deploy.env`:

```yaml
spec:
  deploy:
    env:
      - name: DATABASE_URL
        value: postgres://db.example.com/mydb
      - name: REDIS_URL
        value: redis://redis.default.svc.cluster.local:6379
```

### Secret Variables

Use `valueFrom` to reference Secrets:

```yaml
spec:
  deploy:
    env:
      - name: API_KEY
        valueFrom:
          secretKeyRef:
            name: api-credentials
            key: api-key
```

## Next Steps

- [CRD Specification](function-crd.md) - Function CRD fields and configuration
- [Troubleshooting](troubleshooting.md) - Common issues troubleshooting
- [Git Authentication Guide](../02-guides/git-authentication.md)
- [Registry Configuration](../05-operations/registry-configuration.md)
