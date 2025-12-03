# Troubleshooting

This guide helps diagnose and resolve common issues when using Zenith Operator.

## Useful Commands

```bash
# View Functions
kubectl get functions
kubectl get fn

# View details
kubectl describe function my-function

# View status
kubectl get function my-function -o jsonpath='{.status}'

# View PipelineRuns
kubectl get pipelineruns

# View Knative Services
kubectl get ksvc

# View Triggers
kubectl get triggers

# View operator logs
kubectl logs -n zenith-operator-system deployment/zenith-operator-controller-manager -f

# View function logs
kubectl logs -l serving.knative.dev/service=my-function -f
```

## Common Issues

### Build Failed

**Symptom**: `BuildFailed` condition

**Causes**:
- Git authentication failed
- Buildpack failed to detect language
- Registry push failed

**Solution**:
```bash
# View PipelineRun logs
kubectl get pipelineruns
kubectl logs <pipelinerun-name>-fetch-source-pod --all-containers
```

**Details by Cause**:

#### 1. Git Authentication Failed

**Error message**: `fatal: could not read Username` or `Permission denied (publickey)`

**Checks**:
```bash
# Verify if Secret exists
kubectl get secret github-auth -n your-namespace

# Verify Secret annotations
kubectl get secret github-auth -n your-namespace -o jsonpath='{.metadata.annotations}'

# Verify if ServiceAccount was created
kubectl get serviceaccount <function-name>-sa -n your-namespace

# Verify if Secret is attached to ServiceAccount
kubectl get serviceaccount <function-name>-sa -n your-namespace -o yaml
```

**Solutions**:
- For HTTPS: Verify token has correct permissions (scope `repo` for Classic PAT or `Contents: Read` for Fine-grained)
- For SSH: Verify public key is registered as Deploy Key in GitHub
- Verify `tekton.dev/git-0` annotation matches repository URL

#### 2. Buildpack Failed to Detect Language

**Error message**: `ERROR: No buildpack groups passed detection`

**Causes**:
- Language configuration files not in repository root
- Language not supported by default buildpacks

**Solutions**:
```bash
# Verify repository structure
# Ensure you have at ROOT:
# - Go: go.mod
# - Node.js: package.json
# - Python: requirements.txt or Pipfile
# - Java: pom.xml or build.gradle
```

#### 3. Registry Push Failed

**Error message**: `401 Unauthorized` or `denied: requested access to the resource is denied`

**Checks**:
```bash
# Verify registry secret exists
kubectl get secret registry-credentials -n your-namespace

# Verify if attached to ServiceAccount
kubectl get serviceaccount <function-name>-sa -n your-namespace -o yaml | grep imagePullSecrets
```

**Solutions**:
- Verify registry credentials
- Test manually: `docker login <registry-url> -u <username> -p <password>`
- For private registries, ensure Secret is correctly configured

### Function Not Responding

**Symptom**: Timeout or connection refused

**Causes**:
- Application not listening on correct port
- Application not starting
- Slow scale-from-zero

**Solution**:
```bash
# View pods
kubectl get pods -l serving.knative.dev/service=my-function

# View logs
kubectl logs -l serving.knative.dev/service=my-function
```

**Details by Cause**:

#### 1. Incorrect Port

**Problem**: Application listens on port different from expected

**Check**:
```bash
# View application logs
kubectl logs -l serving.knative.dev/service=my-function | grep -i "listening\|port"
```

**Solution**:
- Ensure your application reads `PORT` environment variable
- Default is `8080`, but Knative might use other ports
- Go Example:
  ```go
  port := os.Getenv("PORT")
  if port == "" {
      port = "8080"
  }
  ```

#### 2. Application Does Not Start

**Problem**: Container crashloop or startup error

**Check**:
```bash
# View pod events
kubectl get pods -l serving.knative.dev/service=my-function
kubectl describe pod <pod-name>

# View full logs
kubectl logs <pod-name> --all-containers
```

**Common Solutions**:
- Check missing dependencies
- Check required environment variables
- Check file permissions
- Check if startup command is correct

#### 3. Slow Scale-from-Zero

**Problem**: First request takes too long (cold start)

**Explanation**: Knative needs to create pod before processing request

**Solutions**:
- Configure higher timeout in HTTP client
- Configure min-scale to keep at least 1 pod:
  ```yaml
  apiVersion: serving.knative.dev/v1
  kind: Service
  metadata:
    name: my-function
  spec:
    template:
      metadata:
        annotations:
          autoscaling.knative.dev/min-scale: "1"
  ```

### Events Do Not Arrive

**Symptom**: Function not receiving events

**Causes**:
- Broker does not exist
- Filters do not match
- Trigger not created

**Solution**:
```bash
# View Broker
kubectl get broker

# View Trigger
kubectl get trigger
kubectl describe trigger my-function-trigger
```

**Details by Cause**:

#### 1. Broker Does Not Exist

**Check**:
```bash
kubectl get broker <broker-name> -n <namespace>
```

**Solution**:
```bash
# Create Broker
cat <<EOF | kubectl apply -f -
apiVersion: eventing.knative.dev/v1
kind: Broker
metadata:
  name: default
  namespace: default
EOF
```

#### 2. Filters Do Not Match

**Problem**: Event attributes do not match Trigger filters

**Check**:
```bash
# View Trigger filters
kubectl get trigger my-function-trigger -o yaml | grep -A 10 filter
```

**Solution**:
- Ensure sent events have correct attributes
- Test by sending event with matching attributes:
  ```bash
  curl -v http://broker-ingress.knative-eventing.svc.cluster.local/default/default \
    -X POST \
    -H "Ce-Id: test-123" \
    -H "Ce-Specversion: 1.0" \
    -H "Ce-Type: com.example.order.created" \
    -H "Ce-Source: payment-service" \
    -H "Content-Type: application/json" \
    -d '{"test": true}'
  ```

#### 3. Trigger Not Created

**Check**:
```bash
kubectl get trigger
kubectl describe function my-function
```

**Solution**:
- Verify `spec.eventing` is configured in Function CR
- Check operator logs for errors:
  ```bash
  kubectl logs -n zenith-operator-system deployment/zenith-operator-controller-manager
  ```

### URL Not Accessible

**Symptom**: Cannot access function URL

**Causes**:
- Envoy Gateway not configured
- HTTPRoute not created
- DNS not resolving

**Solution**:
```bash
# Check Envoy Gateway
kubectl get svc -n envoy-gateway-system

# Check HTTPRoute
kubectl get httproute

# Check Gateway
kubectl get gateway -n knative-serving
```

**Details**:

#### Internal Access (Cluster)

**URL**: `http://<function-name>.<namespace>.svc.cluster.local`

**Test**:
```bash
kubectl run curl-pod --image=curlimages/curl --rm -it --restart=Never -- \
  curl -v http://my-function.default.svc.cluster.local
```

#### External Access (Public)

**Requirements**:
- Envoy Gateway installed
- Gateway configured
- LoadBalancer or NodePort configured

**Test**:
```bash
# Get Envoy Gateway IP
ENVOY_IP=$(kubectl get svc -n envoy-gateway-system \
  -l gateway.envoyproxy.io/owning-gateway-name=knative-gateway \
  -o jsonpath='{.items[0].status.loadBalancer.ingress[0].ip}')

# Get function hostname
FUNCTION_HOST=$(kubectl get function my-function -o jsonpath='{.status.url}' | sed 's|http://||')

# Make request with Host header
curl -H "Host: $FUNCTION_HOST" http://$ENVOY_IP/
```

### LoadBalancer Service in "Pending" State (kind/Minikube)

**Symptom**: Envoy Gateway Service remains in "Pending" state and doesn't receive external IP

**Check**:
```bash
# Check Service status
kubectl get svc -n envoy-gateway-system

# If EXTERNAL-IP shows <pending>, MetalLB is not working
```

**Cause**: Local clusters (kind/Minikube) do not have native LoadBalancer support. MetalLB is required to provide external IPs.

**Solution**:

1. **Check if MetalLB was enabled during installation:**
```bash
# Check if MetalLB is installed
kubectl get pods -n metallb-system

helm upgrade zenith-operator zenith/zenith-operator \
  --namespace zenith-operator-system
```

2. **Check if IPAddressPool was created:**
```bash
kubectl get ipaddresspool -n metallb-system
kubectl get l2advertisement -n metallb-system
```

3. **Check MetalLB logs:**
```bash
kubectl logs -n metallb-system -l app=metallb -c controller
```

> **Note:** On managed clouds (GKE/EKS/AKS), DO NOT enable MetalLB. The cloud native LoadBalancer is used automatically.

### Function Status Shows "GitAuthMissing"

**Symptom**: Condition with reason `GitAuthMissing`

**Cause**: Secret specified in `gitAuthSecretName` does not exist

**Solution**:
```bash
# Verify if Secret exists
kubectl get secret <secret-name> -n <namespace>

# If not exists, create as per documentation
# See: docs/02-guides/git-authentication.md
```

### Dapr Sidecar Not Injecting

**Symptom**: Pod does not have Dapr container

**Causes**:
- Dapr not installed in cluster
- Namespace missing label for Dapr injection
- Incorrect configuration in Function CR

**Check**:
```bash
# Verify Dapr installed
kubectl get pods -n dapr-system

# Verify pod annotations
kubectl get pod <pod-name> -o yaml | grep -A 5 annotations

# Verify if sidecar exists
kubectl get pod <pod-name> -o jsonpath='{.spec.containers[*].name}'
```

**Solutions**:
- Install Dapr: `helm install dapr dapr/dapr --namespace dapr-system`
- Verify Function CR configuration:
  ```yaml
  spec:
    deploy:
      dapr:
        enabled: true
        appID: my-function
        appPort: 8080
  ```

### Image Not Updating

**Symptom**: Function keeps using old image after rebuild

**Cause**: Knative Service was not updated with new digest

**Check**:
```bash
# View imageDigest in Function status
kubectl get function my-function -o jsonpath='{.status.imageDigest}'

# View image in Knative Service
kubectl get ksvc my-function -o jsonpath='{.spec.template.spec.containers[0].image}'
```

**Solution**:
```bash
# Force reconciliation
kubectl annotate function my-function \
  reconcile=$(date +%s) --overwrite

# Or delete and recreate PipelineRun
kubectl delete pipelinerun -l function=my-function
```

## Logs and Debugging

### View Operator Logs

```bash
# Real-time logs
kubectl logs -n zenith-operator-system \
  deployment/zenith-operator-controller-manager -f

# Filtered logs
kubectl logs -n zenith-operator-system \
  deployment/zenith-operator-controller-manager \
  | grep "my-function"
```

### View Build Logs

```bash
# List PipelineRuns
kubectl get pipelineruns

# View git-clone logs
kubectl logs <pipelinerun-name>-fetch-source-pod \
  -c step-clone --tail=50

# View buildpacks logs
kubectl logs <pipelinerun-name>-build-and-push-pod \
  -c step-build --tail=100
```

### View Function Logs

```bash
# Real-time logs
kubectl logs -l serving.knative.dev/service=my-function -f

# All containers logs (including Dapr)
kubectl logs -l serving.knative.dev/service=my-function \
  --all-containers=true

# Specific pod logs
kubectl logs <pod-name> -c user-container
```

### Interactive Debugging

```bash
# Run shell in function pod
kubectl exec -it <pod-name> -c user-container -- /bin/sh

# Test connectivity from inside pod
kubectl exec -it <pod-name> -c user-container -- \
  curl http://other-service.default.svc.cluster.local
```

## Configuration Validation

### Validate Function CR

```bash
# Validate YAML syntax
kubectl apply --dry-run=client -f function.yaml

# Validate with server (includes schema validation)
kubectl apply --dry-run=server -f function.yaml
```

### Validate Secrets

```bash
# Check Git Secret
kubectl get secret github-auth -o yaml

# Check annotations
kubectl get secret github-auth -o jsonpath='{.metadata.annotations}'

# Check content (base64 decoded)
kubectl get secret github-auth -o jsonpath='{.data.password}' | base64 -d
```

### Validate ServiceAccount

```bash
# View ServiceAccount
kubectl get serviceaccount <function-name>-sa -o yaml

# Check attached secrets
kubectl get serviceaccount <function-name>-sa \
  -o jsonpath='{.secrets[*].name}'

# Check imagePullSecrets
kubectl get serviceaccount <function-name>-sa \
  -o jsonpath='{.imagePullSecrets[*].name}'
```

## Additional Resources

- [CRD Specification](function-crd.md) - Detailed fields and configuration
- [Operator Reference](operator-reference.md) - Behavior and integrations
- [Git Authentication Guide](../02-guides/git-authentication.md) - Authentication setup
- [Registry Configuration](../05-operations/registry-configuration.md) - Registry setup

## Getting Help

If you cannot resolve the issue:

1. **Collect information**:
   ```bash
   # Save Function status
   kubectl get function my-function -o yaml > function-status.yaml
   
   # Save operator logs
   kubectl logs -n zenith-operator-system \
     deployment/zenith-operator-controller-manager \
     --tail=200 > operator-logs.txt
   
   # Save PipelineRun logs
   kubectl logs <pipelinerun-name>-fetch-source-pod \
     --all-containers > build-logs.txt
   ```

2. **Open an issue**: https://github.com/LucasGois1/zenith-operator/issues

3. **Include**:
   - Problem description
   - Operator version
   - Function CR (without sensitive info)
   - Relevant logs
   - Reproduction steps
