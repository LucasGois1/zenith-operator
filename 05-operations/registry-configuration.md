# Registry Configuration for Production

This document explains how to configure the Zenith Operator to work with Docker Hub or custom container registries in production environments.

## Overview

The Zenith Operator uses Cloud Native Buildpacks to build container images from source code. During the build process, buildpacks need to access a container registry to:

1. **Analyze phase**: Check if previous image layers exist (for optimization)
2. **Export phase**: Push the built image to the registry

## Smart Registry Detection

**The operator automatically detects registry types and configures insecure registries when needed.** You don't need to configure anything for most common scenarios:

### Automatic Detection (No Configuration Needed)

The operator automatically detects and configures insecure registries for:

1. **Cluster-internal registries**: Any registry with `.svc.cluster.local` in the hostname
   - Example: `registry.registry.svc.cluster.local:5000`
   - Automatically detected and configured as insecure

2. **Localhost registries**: Registries running on localhost
   - Examples: `localhost:5000`, `127.0.0.1:5000`
   - Automatically detected and configured as insecure

3. **Development registries**: Registries with non-standard ports (except known public registries)
   - Example: `my-registry.local:5000`
   - Automatically detected and configured as insecure

4. **Public registries**: Docker Hub, GCR, GHCR, Quay, etc.
   - Examples: `docker.io/myuser/myapp`, `gcr.io/myproject/myapp`
   - Automatically use HTTPS (no insecure configuration needed)

### Manual Configuration (Optional)

If you need to override the automatic detection or add additional insecure registries, use the `INSECURE_REGISTRIES` environment variable in the operator deployment:

```yaml
# config/manager/manager.yaml
env:
  - name: INSECURE_REGISTRIES
    value: "registry1.example.com:5000,registry2.example.com:5000"
```

**When to use manual configuration:**
- You have a custom registry that isn't automatically detected
- You want to disable automatic detection (set to empty string)
- You need to add multiple custom registries

## Production Configuration Options

### Option 1: Docker Hub (Recommended for Getting Started)

Docker Hub is the easiest option for getting started. You'll need:

1. A Docker Hub account
2. Registry credentials configured in Kubernetes

#### Step 1: Create Docker Hub Credentials Secret

```bash
# Create a secret with your Docker Hub credentials
kubectl create secret docker-registry dockerhub-credentials \
  --docker-server=https://index.docker.io/v1/ \
  --docker-username=<your-dockerhub-username> \
  --docker-password=<your-dockerhub-password> \
  --docker-email=<your-email> \
  -n <function-namespace>
```

#### Step 2: Configure the Operator

**Option A: Environment Variable (Recommended)**

Modify the operator deployment to accept registry configuration via environment variables:

```yaml
# config/manager/manager.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: controller-manager
spec:
  template:
    spec:
      containers:
      - name: manager
        env:
        - name: CNB_INSECURE_REGISTRIES
          value: ""  # Empty for Docker Hub (uses HTTPS)
        - name: REGISTRY_SECRET_NAME
          value: "dockerhub-credentials"
```

**Option B: Modify the Operator Code**

Update `internal/controller/function_controller.go` to remove the hardcoded insecure registry:

```go
// Before (test-only):
{Name: "CNB_INSECURE_REGISTRIES", Value: tektonv1.ParamValue{
    Type: tektonv1.ParamTypeString, 
    StringVal: "registry.registry.svc.cluster.local:5000"
}},

// After (production):
// Remove the CNB_INSECURE_REGISTRIES parameter entirely for Docker Hub
// Or make it configurable via environment variable
```

#### Step 3: Update Function CRD

Ensure your Function resources use Docker Hub image names:

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: my-function
spec:
  build:
    image: docker.io/<your-username>/my-function  # Docker Hub format
  gitRepo:
    url: https://github.com/your-org/your-repo
    revision: main
```

#### Step 4: Configure ServiceAccount with Registry Credentials

The operator automatically creates a ServiceAccount for each Function. You need to link the registry credentials to this ServiceAccount:

**Option A: Automatic (Recommended)**

Modify the operator to automatically attach registry credentials to ServiceAccounts:

```go
// In internal/controller/function_controller.go, update buildServiceAccount():

func (r *FunctionReconciler) buildServiceAccount(function *functionsv1alpha1.Function) *v1.ServiceAccount {
    sa := &v1.ServiceAccount{
        ObjectMeta: metav1.ObjectMeta{
            Name:      function.Name + "-sa",
            Namespace: function.Namespace,
        },
    }
    
    // Add registry credentials
    registrySecretName := os.Getenv("REGISTRY_SECRET_NAME")
    if registrySecretName != "" {
        sa.ImagePullSecrets = append(sa.ImagePullSecrets, v1.LocalObjectReference{
            Name: registrySecretName,
        })
    }
    
    return sa
}
```

**Option B: Manual**

Patch the ServiceAccount after the Function is created:

```bash
kubectl patch serviceaccount <function-name>-sa \
  -n <namespace> \
  -p '{"imagePullSecrets":[{"name":"dockerhub-credentials"}]}'
```

### Option 2: Custom Container Registry

If you're using a custom registry (Harbor, Nexus, AWS ECR, GCR, etc.), follow similar steps:

#### Step 1: Create Registry Credentials Secret

```bash
kubectl create secret docker-registry custom-registry-credentials \
  --docker-server=<your-registry-url> \
  --docker-username=<username> \
  --docker-password=<password> \
  --docker-email=<email> \
  -n <function-namespace>
```

#### Step 2: Configure Insecure Registry (if needed)

If your registry uses self-signed certificates or HTTP (not recommended for production):

```yaml
# In operator deployment
env:
- name: CNB_INSECURE_REGISTRIES
  value: "your-registry.example.com:5000"
```

Or modify the operator code:

```go
// Read from environment variable
insecureRegistries := os.Getenv("CNB_INSECURE_REGISTRIES")
if insecureRegistries != "" {
    params = append(params, tektonv1.Param{
        Name: "CNB_INSECURE_REGISTRIES",
        Value: tektonv1.ParamValue{
            Type: tektonv1.ParamTypeString,
            StringVal: insecureRegistries,
        },
    })
}
```

#### Step 3: Update Function CRD

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: my-function
spec:
  build:
    image: your-registry.example.com/my-org/my-function
  gitRepo:
    url: https://github.com/your-org/your-repo
    revision: main
```

### Option 3: In-Cluster Registry (Production)

For production use of an in-cluster registry, you should:

1. **Use TLS certificates** (not insecure HTTP)
2. **Configure authentication**
3. **Set up persistent storage**
4. **Configure backup and disaster recovery**

Example production registry setup:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: registry
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: registry-tls
  namespace: registry
spec:
  secretName: registry-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - registry.example.com
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: registry
  namespace: registry
spec:
  replicas: 2  # High availability
  selector:
    matchLabels:
      app: registry
  template:
    metadata:
      labels:
        app: registry
    spec:
      containers:
      - name: registry
        image: registry:2
        env:
        - name: REGISTRY_AUTH
          value: "htpasswd"
        - name: REGISTRY_AUTH_HTPASSWD_PATH
          value: "/auth/htpasswd"
        - name: REGISTRY_AUTH_HTPASSWD_REALM
          value: "Registry Realm"
        - name: REGISTRY_HTTP_TLS_CERTIFICATE
          value: "/certs/tls.crt"
        - name: REGISTRY_HTTP_TLS_KEY
          value: "/certs/tls.key"
        ports:
        - containerPort: 5000
        volumeMounts:
        - name: registry-storage
          mountPath: /var/lib/registry
        - name: auth
          mountPath: /auth
        - name: certs
          mountPath: /certs
      volumes:
      - name: registry-storage
        persistentVolumeClaim:
          claimName: registry-pvc
      - name: auth
        secret:
          secretName: registry-auth
      - name: certs
        secret:
          secretName: registry-tls
```

## Recommended Implementation: Make Registry Configurable

The best approach is to make the registry configuration flexible via environment variables or CRD fields:

### Approach 1: Environment Variables (Simplest)

```go
// internal/controller/function_controller.go

func (r *FunctionReconciler) buildPipelineRun(function *functionsv1alpha1.Function) *tektonv1.PipelineRun {
    // ... existing code ...
    
    params := []tektonv1.Param{
        {Name: "APP_IMAGE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: function.Spec.Build.Image}},
        {Name: "CNB_BUILDER_IMAGE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "paketobuildpacks/builder-jammy-base:latest"}},
        {Name: "CNB_PROCESS_TYPE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
    }
    
    // Add insecure registries if configured
    insecureRegistries := os.Getenv("CNB_INSECURE_REGISTRIES")
    if insecureRegistries != "" {
        params = append(params, tektonv1.Param{
            Name: "CNB_INSECURE_REGISTRIES",
            Value: tektonv1.ParamValue{
                Type: tektonv1.ParamTypeString,
                StringVal: insecureRegistries,
            },
        })
    }
    
    // ... rest of the code ...
}
```

Then configure via deployment:

```yaml
# config/manager/manager.yaml
env:
- name: CNB_INSECURE_REGISTRIES
  value: ""  # Empty for secure registries (Docker Hub, GCR, etc.)
  # Or set to "registry.example.com:5000" for insecure registries
```

### Approach 2: CRD Field (Most Flexible)

Add registry configuration to the Function CRD:

```go
// api/v1alpha1/function_types.go

type BuildSpec struct {
    Image string `json:"image"`
    
    // Registry configuration (optional)
    Registry *RegistryConfig `json:"registry,omitempty"`
}

type RegistryConfig struct {
    // Insecure registries that don't use HTTPS
    InsecureRegistries []string `json:"insecureRegistries,omitempty"`
    
    // Secret name containing registry credentials
    CredentialsSecret string `json:"credentialsSecret,omitempty"`
}
```

Then use it in the controller:

```go
func (r *FunctionReconciler) buildPipelineRun(function *functionsv1alpha1.Function) *tektonv1.PipelineRun {
    // ... existing code ...
    
    params := []tektonv1.Param{
        {Name: "APP_IMAGE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: function.Spec.Build.Image}},
        {Name: "CNB_BUILDER_IMAGE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: "paketobuildpacks/builder-jammy-base:latest"}},
        {Name: "CNB_PROCESS_TYPE", Value: tektonv1.ParamValue{Type: tektonv1.ParamTypeString, StringVal: ""}},
    }
    
    // Add insecure registries from Function spec
    if function.Spec.Build.Registry != nil && len(function.Spec.Build.Registry.InsecureRegistries) > 0 {
        insecureRegistries := strings.Join(function.Spec.Build.Registry.InsecureRegistries, ",")
        params = append(params, tektonv1.Param{
            Name: "CNB_INSECURE_REGISTRIES",
            Value: tektonv1.ParamValue{
                Type: tektonv1.ParamTypeString,
                StringVal: insecureRegistries,
            },
        })
    }
    
    // ... rest of the code ...
}
```

## Testing Your Configuration

After configuring the registry, test with a simple Function:

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: test-function
  namespace: default
spec:
  build:
    image: docker.io/<your-username>/test-function
  gitRepo:
    url: https://github.com/LucasGois1/zenith-test-functions
    revision: main
  deploy: {}
```

Check the build:

```bash
# Watch the PipelineRun
kubectl get pipelineruns -w

# Check build logs
kubectl logs -f <pipelinerun-name>-build-and-push-pod -c step-export

# Verify image was pushed
docker pull docker.io/<your-username>/test-function@<digest>
```

## Troubleshooting

### Build fails with "401 Unauthorized"

**Cause**: Registry credentials are not configured or incorrect.

**Solution**:
1. Verify the secret exists: `kubectl get secret <secret-name>`
2. Check the secret is attached to the ServiceAccount: `kubectl get sa <function-name>-sa -o yaml`
3. Verify credentials are correct by testing manually:
   ```bash
   docker login <registry-url> -u <username> -p <password>
   ```

### Build fails with "x509: certificate signed by unknown authority"

**Cause**: Registry uses self-signed certificates.

**Solution**:
1. Add the registry to insecure registries (not recommended for production)
2. Or add the CA certificate to the cluster's trust store

### Build fails with "connection refused"

**Cause**: Registry is not accessible from the cluster.

**Solution**:
1. Verify registry is accessible: `kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- curl -v <registry-url>`
2. Check network policies and firewall rules
3. Verify DNS resolution

## Migration from Test to Production

When moving from the test environment to production:

1. **Remove the local registry** from your cluster (or keep it for development)
2. **Update the operator** to use environment variables or CRD fields for registry configuration
3. **Create registry credentials secrets** in all namespaces where Functions will be deployed
4. **Update Function CRDs** to use production registry URLs
5. **Test thoroughly** with a non-critical Function before deploying production workloads

## Security Best Practices

1. **Always use HTTPS** for registry communication in production
2. **Use strong passwords** or token-based authentication
3. **Rotate credentials regularly**
4. **Use separate registries** for development, staging, and production
5. **Implement image scanning** for vulnerabilities
6. **Use private registries** for proprietary code
7. **Limit registry access** using Kubernetes RBAC and network policies
8. **Enable audit logging** for registry access

## Additional Resources

- [Cloud Native Buildpacks Documentation](https://buildpacks.io/docs/)
- [Tekton Pipeline Documentation](https://tekton.dev/docs/pipelines/)
- [Docker Registry Documentation](https://docs.docker.com/registry/)
- [Kubernetes Secrets Documentation](https://kubernetes.io/docs/concepts/configuration/secret/)
