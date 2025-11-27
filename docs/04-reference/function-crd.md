# Function CRD - Complete Specification

This documentation describes the complete specification of the `Function` Custom Resource Definition (CRD).

## API Group and Version

- **API Group**: `functions.zenith.com`
- **Version**: `v1alpha1`
- **Kind**: `Function`
- **Plural**: `functions`
- **Singular**: `function`
- **Short Names**: `fn`, `func`

## Complete Example

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: example-function
  namespace: default
  labels:
    app: my-app
    environment: production
  annotations:
    description: "Example function with all features"
spec:
  # Git Configuration
  gitRepo: https://github.com/myorg/my-function
  gitRevision: main
  gitAuthSecretName: github-auth
  
  # Build Configuration
  build:
    image: registry.example.com/my-function:latest
    registrySecretName: registry-credentials
  
  # Deploy Configuration
  deploy:
    dapr:
      enabled: true
      appID: example-function
      appPort: 8080
    env:
      - name: DATABASE_URL
        value: postgres://db.example.com/mydb
      - name: LOG_LEVEL
        value: info
  
  # Eventing Configuration (optional)
  eventing:
    broker: default
    filters:
      type: com.example.event.created
      source: my-service
status:
  # Populated by operator
  conditions:
    - type: Ready
      status: "True"
      reason: FunctionReady
      message: Function is ready
  imageDigest: registry.example.com/my-function@sha256:abc123...
  url: http://example-function.default.svc.cluster.local
  observedGeneration: 1
```

## Spec Fields

### gitRepo (Required)

**Type**: `string`

**Description**: URL of the Git repository containing the function source code.

**Supported Protocols**:
- HTTPS: `https://github.com/myorg/my-function`
- SSH: `git@github.com:myorg/my-function.git`

**Examples**:
```yaml
# GitHub HTTPS
gitRepo: https://github.com/myorg/my-function

# GitLab HTTPS
gitRepo: https://gitlab.com/myorg/my-function

# GitHub SSH
gitRepo: git@github.com:myorg/my-function.git

# Self-hosted
gitRepo: https://git.example.com/myorg/my-function
```

### gitRevision (Optional)

**Type**: `string`

**Default**: `main`

**Description**: Git revision to use (branch, tag, or commit hash).

**Examples**:
```yaml
# Branch
gitRevision: main
gitRevision: develop
gitRevision: feature/new-feature

# Tag
gitRevision: v1.0.0
gitRevision: release-2024-01

# Commit hash
gitRevision: abc123def456
gitRevision: 1234567890abcdef1234567890abcdef12345678
```

### gitAuthSecretName (Optional)

**Type**: `string`

**Description**: Name of the Secret used to authenticate with private Git repository.

**Secret Type**: `kubernetes.io/basic-auth` or `kubernetes.io/ssh-auth`

**Required Annotation**: `tekton.dev/git-0: <git-server>`

**Examples**:
```yaml
# For private repositories
gitAuthSecretName: github-auth
gitAuthSecretName: gitlab-credentials
```

**Secret Example**:
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
  password: ghp_mytoken
```

### build (Required)

**Type**: `BuildSpec`

**Description**: Build pipeline configuration.

#### build.image (Required)

**Type**: `string`

**Description**: Full name of target image (without tag or digest).

**Format**: `<registry>/<repository>/<image>`

**Examples**:
```yaml
# Docker Hub
image: docker.io/myorg/my-function

# GitHub Container Registry
image: ghcr.io/myorg/my-function

# Google Container Registry
image: gcr.io/myproject/my-function

# Azure Container Registry
image: myregistry.azurecr.io/my-function

# Local registry
image: registry.registry.svc.cluster.local:5000/my-function
```

**Note**: The operator automatically adds the digest after build: `image@sha256:...`

#### build.registrySecretName (Optional)

**Type**: `string`

**Description**: Name of the Secret used to authenticate with container registry.

**Secret Type**: `kubernetes.io/dockerconfigjson`

**Examples**:
```yaml
registrySecretName: registry-credentials
registrySecretName: dockerhub-secret
```

**Secret Example**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: registry-credentials
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: <base64-encoded-docker-config>
```

### deploy (Required)

**Type**: `DeploySpec`

**Description**: Function deployment configuration.

#### deploy.dapr (Optional)

**Type**: `DaprConfig`

**Description**: Dapr sidecar configuration.

##### deploy.dapr.enabled (Required if dapr specified)

**Type**: `boolean`

**Description**: If `true`, injects Dapr sidecar into the pod.

**Default**: `false`

**Examples**:
```yaml
dapr:
  enabled: true
```

##### deploy.dapr.appID (Required if dapr.enabled=true)

**Type**: `string`

**Description**: Unique Dapr application ID.

**Constraints**:
- Must be unique within namespace
- Lowercase alphanumeric and hyphens
- Maximum 63 characters

**Examples**:
```yaml
dapr:
  enabled: true
  appID: my-function
  appID: order-processor
  appID: payment-service
```

##### deploy.dapr.appPort (Required if dapr.enabled=true)

**Type**: `integer`

**Description**: Port where application listens.

**Default**: None (must be specified)

**Common Values**: `8080`, `3000`, `8000`

**Examples**:
```yaml
dapr:
  enabled: true
  appID: my-function
  appPort: 8080
```

#### deploy.env (Optional)

**Type**: `[]corev1.EnvVar`

**Description**: List of environment variables to inject into function container. Supports static values, references to Secrets/ConfigMaps, and Pod fields.

**EnvVar Fields**:
- `name` (string, required): Name of environment variable
- `value` (string, optional): Static value
- `valueFrom` (object, optional): Source for value (cannot be used with `value`)
  - `secretKeyRef`: Reference to a key in a Secret
    - `name` (string): Secret name
    - `key` (string): Key inside Secret
    - `optional` (boolean): If true, does not fail if Secret does not exist
  - `configMapKeyRef`: Reference to a key in a ConfigMap
    - `name` (string): ConfigMap name
    - `key` (string): Key inside ConfigMap
    - `optional` (boolean): If true, does not fail if ConfigMap does not exist
  - `fieldRef`: Reference to a Pod field
    - `fieldPath` (string): Field path (e.g., metadata.name, metadata.namespace)
  - `resourceFieldRef`: Reference to container resources
    - `resource` (string): Resource name (e.g., limits.cpu, requests.memory)

**Examples**:

**Static values**:
```yaml
env:
  - name: DATABASE_URL
    value: postgres://db.example.com/mydb
  - name: LOG_LEVEL
    value: debug
  - name: FEATURE_FLAG_X
    value: "true"
```

**References to Secrets**:
```yaml
env:
  - name: API_KEY
    valueFrom:
      secretKeyRef:
        name: api-credentials
        key: api-key
  - name: DATABASE_PASSWORD
    valueFrom:
      secretKeyRef:
        name: db-credentials
        key: password
  - name: OPTIONAL_TOKEN
    valueFrom:
      secretKeyRef:
        name: optional-secret
        key: token
        optional: true
```

**References to ConfigMaps**:
```yaml
env:
  - name: APP_CONFIG
    valueFrom:
      configMapKeyRef:
        name: app-config
        key: config.json
  - name: FEATURE_FLAGS
    valueFrom:
      configMapKeyRef:
        name: feature-flags
        key: flags
```

**References to Pod fields**:
```yaml
env:
  - name: POD_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.name
  - name: POD_NAMESPACE
    valueFrom:
      fieldRef:
        fieldPath: metadata.namespace
  - name: POD_IP
    valueFrom:
      fieldRef:
        fieldPath: status.podIP
```

**References to resources**:
```yaml
env:
  - name: CPU_LIMIT
    valueFrom:
      resourceFieldRef:
        resource: limits.cpu
  - name: MEMORY_REQUEST
    valueFrom:
      resourceFieldRef:
        resource: requests.memory
```

**Combined example**:
```yaml
env:
  - name: APP_ENV
    value: production
  - name: DATABASE_PASSWORD
    valueFrom:
      secretKeyRef:
        name: db-credentials
        key: password
  - name: CONFIG_PATH
    valueFrom:
      configMapKeyRef:
        name: app-config
        key: config-path
  - name: POD_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.name
```

**Important Notes**:
- The operator validates that referenced Secrets and ConfigMaps exist before deploying the function
- If a Secret or ConfigMap does not exist, function status will be `Ready=False` with reason `SecretNotFound` or `ConfigMapNotFound`
- Use `optional: true` for resources that may not exist
- Changes to environment variables trigger a new Knative Service revision

#### deploy.envFrom (Optional)

**Type**: `[]corev1.EnvFromSource`

**Description**: List of sources to populate environment variables in container. All keys from Secret or ConfigMap will be exposed as environment variables.

**EnvFromSource Fields**:
- `secretRef`: Reference to a Secret
  - `name` (string): Secret name
  - `optional` (boolean): If true, does not fail if Secret does not exist
- `configMapRef`: Reference to a ConfigMap
  - `name` (string): ConfigMap name
  - `optional` (boolean): If true, does not fail if ConfigMap does not exist
- `prefix` (string, optional): Prefix to add to variable names

**Examples**:

**Inject all keys from a Secret**:
```yaml
envFrom:
  - secretRef:
      name: api-credentials
```

**Inject all keys from a ConfigMap**:
```yaml
envFrom:
  - configMapRef:
      name: app-config
```

**With prefix**:
```yaml
envFrom:
  - prefix: DB_
    secretRef:
      name: database-credentials
  - prefix: CACHE_
    configMapRef:
      name: redis-config
```

**Multiple sources**:
```yaml
envFrom:
  - secretRef:
      name: api-credentials
  - configMapRef:
      name: app-config
  - secretRef:
      name: optional-secrets
      optional: true
```

**Combined example with env**:
```yaml
deploy:
  env:
    - name: APP_ENV
      value: production
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
  envFrom:
    - secretRef:
        name: api-credentials
    - configMapRef:
        name: app-config
```

#### deploy.scale (Optional)

**Type**: `ScaleSpec`

**Description**: Autoscaling configuration to control minimum and maximum replicas.

##### deploy.scale.minScale (Optional)

**Type**: `integer`

**Description**: Minimum number of replicas to maintain. If set to 1 or more, prevents scale-to-zero and eliminates cold starts.

**Default**: `0` (scale-to-zero enabled)

**Minimum**: `0`

**Examples**:
```yaml
deploy:
  scale:
    minScale: 1  # Always keep 1 replica active
```

##### deploy.scale.maxScale (Optional)

**Type**: `integer`

**Description**: Maximum number of replicas allowed. If set to 0, no maximum limit.

**Default**: `0` (no limit)

**Minimum**: `0`

**Examples**:
```yaml
deploy:
  scale:
    maxScale: 10  # Limits to 10 replicas maximum
```

**Combined Example**:
```yaml
deploy:
  scale:
    minScale: 1    # Eliminates cold starts
    maxScale: 50   # Controls maximum costs
  env:
    - name: LOG_LEVEL
      value: info
```

**Use Cases**:
- **Critical APIs**: Use `minScale: 1` to eliminate cold starts
- **Cost control**: Use `maxScale` to limit maximum resources
- **Development**: Keep defaults (scale-to-zero) to save resources

**Knative Annotations**:
When configured, the operator adds the following annotations to Knative Service template:
- `autoscaling.knative.dev/min-scale`: Value of minScale
- `autoscaling.knative.dev/max-scale`: Value of maxScale

### eventing (Optional)

**Type**: `EventingSpec`

**Description**: Event subscription configuration via Knative Eventing.

**Note**: If specified, the operator creates a Knative Trigger.

#### eventing.broker (Optional)

**Type**: `string`

**Default**: `default`

**Description**: Name of Knative Broker to subscribe to.

**Examples**:
```yaml
eventing:
  broker: default
  broker: production
  broker: staging
```

**Note**: Broker must exist in the same namespace.

#### eventing.filters (Optional)

**Type**: `map[string]string`

**Description**: Map of CloudEvents attributes to filter events.

**Common Attributes**:
- `type`: Event type
- `source`: Event source
- `subject`: Event subject
- Custom attributes

**Examples**:
```yaml
# Filter by type
eventing:
  broker: default
  filters:
    type: com.example.order.created

# Filter by type and source
eventing:
  broker: default
  filters:
    type: com.example.order.created
    source: payment-service

# Multiple filters (AND)
eventing:
  broker: default
  filters:
    type: com.example.order.created
    source: payment-service
    subject: orders

# No filters (all events)
eventing:
  broker: default
  filters: {}
```

**Note**: All filters must match (AND operation).

### observability (Optional)

**Type**: `ObservabilitySpec`

**Description**: Observability and distributed tracing configuration via OpenTelemetry.

**Note**: If specified, the operator injects OpenTelemetry environment variables into container.

#### observability.tracing (Optional)

**Type**: `TracingConfig`

**Description**: Distributed tracing configuration.

##### observability.tracing.enabled (Required if tracing specified)

**Type**: `boolean`

**Description**: If `true`, enables distributed tracing via OpenTelemetry.

**Default**: `false`

**Examples**:
```yaml
observability:
  tracing:
    enabled: true
```

**Behavior**:
- When enabled, the operator automatically injects OpenTelemetry environment variables into container
- If Dapr is also enabled, the operator configures Dapr to propagate trace context

**Environment Variables Injected**:
- `OTEL_EXPORTER_OTLP_ENDPOINT`: Endpoint of OpenTelemetry Collector
- `OTEL_SERVICE_NAME`: Function name (used to identify service in traces)
- `OTEL_RESOURCE_ATTRIBUTES`: Resource attributes (namespace, version)
- `OTEL_TRACES_EXPORTER`: Export protocol (otlp)
- `OTEL_TRACES_SAMPLER`: Sampler type (if samplingRate specified)
- `OTEL_TRACES_SAMPLER_ARG`: Sampler argument (if samplingRate specified)

##### observability.tracing.samplingRate (Optional)

**Type**: `string`

**Description**: Trace sampling rate (0.0 to 1.0). If not specified, uses OpenTelemetry Collector default rate.

**Format**: String representing a decimal number between 0.0 and 1.0

**Validation**: Must match regex pattern `^(0(\.\d+)?|1(\.0+)?)$`

**Examples**:
```yaml
# 100% sampling (captures all traces)
observability:
  tracing:
    enabled: true
    samplingRate: "1.0"

# 50% sampling
observability:
  tracing:
    enabled: true
    samplingRate: "0.5"

# 10% sampling
observability:
  tracing:
    enabled: true
    samplingRate: "0.1"

# 1% sampling
observability:
  tracing:
    enabled: true
    samplingRate: "0.01"
```

**Recommendations**:
- **Development**: Use `"1.0"` (100%) to capture all traces
- **Staging**: Use `"0.5"` to `"1.0"` (50-100%) for critical functions
- **Production**: Use `"0.01"` to `"0.1"` (1-10%) for high traffic functions

**Example with Dapr**:
```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: payment-processor
spec:
  gitRepo: https://github.com/myorg/payment-processor
  gitRevision: main
  build:
    image: registry.example.com/payment-processor:latest
  deploy:
    dapr:
      enabled: true
      appID: payment-processor
      appPort: 8080
  observability:
    tracing:
      enabled: true
      samplingRate: "0.1"
```

**Note**: When Dapr and tracing are enabled, the operator automatically adds annotation `dapr.io/config: tracing-config` to pod template.

## Status Fields

The operator automatically updates the `status` field of the Function.

### conditions

**Type**: `[]metav1.Condition`

**Description**: List of conditions describing function state.

**Condition Types**:
- `Ready`: Indicates if function is ready to receive requests
- `BuildSucceeded`: Indicates if build was successful
- `DeploySucceeded`: Indicates if deploy was successful

**Condition Fields**:
- `type` (string): Condition type
- `status` (string): `True`, `False`, or `Unknown`
- `reason` (string): Machine-readable reason
- `message` (string): Human-readable message
- `lastTransitionTime` (timestamp): Last time condition changed

**Examples**:
```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: FunctionReady
      message: Function is ready and serving traffic
      lastTransitionTime: "2025-01-15T10:30:00Z"
    
    - type: BuildSucceeded
      status: "True"
      reason: BuildCompleted
      message: Image built successfully
      lastTransitionTime: "2025-01-15T10:25:00Z"
```

### imageDigest

**Type**: `string`

**Description**: Immutable reference of built image (with SHA256 digest).

**Format**: `<registry>/<repository>/<image>@sha256:<hash>`

**Examples**:
```yaml
imageDigest: registry.example.com/my-function@sha256:abc123def456...
imageDigest: docker.io/myorg/my-function@sha256:1234567890abcdef...
```

**Note**: Populated after successful build.

### url

**Type**: `string`

**Description**: Publicly accessible URL of the function (from Knative Service).

**Format**: `http://<function-name>.<namespace>.<domain>`

**Examples**:
```yaml
# Internal URL (cluster)
url: http://my-function.default.svc.cluster.local

# External URL (public)
url: http://my-function.default.example.com
```

**Note**: Populated after successful deploy.

### observedGeneration

**Type**: `integer`

**Description**: Spec generation observed by the operator.

**Usage**: Used to detect if operator has processed latest spec change.

**Examples**:
```yaml
observedGeneration: 1
observedGeneration: 5
```

## Status Conditions

### Status Progression

The operator updates conditions as function progresses:

#### 1. Initial State

```yaml
status:
  conditions: []
  observedGeneration: 0
```

#### 2. Building

```yaml
status:
  conditions:
    - type: Ready
      status: "False"
      reason: Building
      message: Building container image
  observedGeneration: 1
```

#### 3. Build Succeeded

```yaml
status:
  conditions:
    - type: Ready
      status: "False"
      reason: BuildSucceeded
      message: Image built successfully, deploying...
    - type: BuildSucceeded
      status: "True"
      reason: BuildCompleted
      message: Image built successfully
  imageDigest: registry.example.com/my-function@sha256:abc123...
  observedGeneration: 1
```

#### 4. Ready

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: FunctionReady
      message: Function is ready and serving traffic
    - type: BuildSucceeded
      status: "True"
      reason: BuildCompleted
      message: Image built successfully
  imageDigest: registry.example.com/my-function@sha256:abc123...
  url: http://my-function.default.svc.cluster.local
  observedGeneration: 1
```

#### 5. Build Failed

```yaml
status:
  conditions:
    - type: Ready
      status: "False"
      reason: BuildFailed
      message: "Build failed: git clone authentication failed"
    - type: BuildSucceeded
      status: "False"
      reason: BuildFailed
      message: "Git clone authentication failed"
  observedGeneration: 1
```

## Next Steps

- [Operator Reference](operator-reference.md) - Operator behavior and integrations
- [Troubleshooting](troubleshooting.md) - Common issues troubleshooting
- [HTTP Functions Guide](../02-guides/http-functions.md)
- [Event Functions Guide](../02-guides/event-functions.md)
