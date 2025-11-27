# Quick Start

This guide shows how to create and deploy your first serverless function with Zenith Operator in 5 minutes.

## Prerequisites

Before starting, ensure you have:

- Kubernetes cluster running (kind, minikube, GKE, EKS, etc.)
- Zenith Operator installed ([see installation guide](installation.md))
- `kubectl` configured to access your cluster
- Container registry accessible (Docker Hub, GCR, etc.)

## Step 1: Create Git Authentication Secret (Optional)

If you are using a private Git repository, create a Secret for authentication:

```bash
kubectl create secret generic github-auth \
  --from-literal=username=myuser \
  --from-literal=password=mytoken \
  --type=kubernetes.io/basic-auth

kubectl annotate secret github-auth \
  tekton.dev/git-0=https://github.com
```

**Note**: For public repositories, you can skip this step.

## Step 2: Create Your First Function

Create a file `my-first-function.yaml` with the following content:

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: my-first-function
spec:
  # Git repository with function code
  gitRepo: https://github.com/LucasGois1/zenith-test-functions
  gitRevision: main
  
  # Authentication secret (remove if using public repository)
  gitAuthSecretName: github-auth
  
  # Build configuration
  build:
    image: registry.example.com/my-first-function:latest
  
  # Deploy configuration
  deploy: {}
```

**Important**: Replace `registry.example.com` with your registry (e.g., `docker.io/myuser`).

Apply the resource:

```bash
kubectl apply -f my-first-function.yaml
```

## Step 3: Check Status

Monitor the function progress:

```bash
# View all functions
kubectl get functions

# View function details
kubectl describe function my-first-function

# View PipelineRun (build)
kubectl get pipelineruns

# View build logs
kubectl logs -f <pipelinerun-name>-fetch-source-pod --all-containers
```

The function status will go through these phases:

1. **Building** - Building container image
2. **BuildSucceeded** - Build completed successfully
3. **Ready** - Function deployed and ready to receive requests

## Step 4: Access the Function

After the function is ready (status `Ready`), you can access it:

```bash
# Get function URL
FUNCTION_URL=$(kubectl get function my-first-function -o jsonpath='{.status.url}')
echo "Function URL: $FUNCTION_URL"

# Make a request
curl $FUNCTION_URL
```

**Expected response**:
```json
{
  "status": "ok",
  "message": "Hello from Zenith Function!",
  "type": "http-sync"
}
```

## Step 5: Update the Function

To update the function, make changes to the code and push to Git, then force a rebuild:

```bash
# Option 1: Add annotation to force rebuild
kubectl annotate function my-first-function \
  rebuild=$(date +%s) --overwrite

# Option 2: Update Git revision
kubectl patch function my-first-function \
  --type merge \
  -p '{"spec":{"gitRevision":"v2.0.0"}}'
```

## Step 6: Clean Resources

To remove the function:

```bash
kubectl delete function my-first-function
```

The operator automatically removes all related resources (PipelineRun, Knative Service, etc.).

## Next Steps

Now that you have created your first function, explore more advanced features:

### Synchronous HTTP Functions
Learn how to create REST APIs and webhooks:
- [HTTP Functions Guide](../02-guides/http-functions.md)

### Asynchronous Functions with Events
Create event-driven functions:
- [Event Functions Guide](../02-guides/event-functions.md)

### Function Communication
Implement microservices architectures:
- [Function Communication Guide](../02-guides/function-communication.md)

### Advanced Configurations
Explore all configuration options:
- [Complete CRD Specification](../04-reference/function-crd.md)
- [Operator Reference](../04-reference/operator-reference.md)

## Troubleshooting

### Build Failed

If the build fails, check the logs:

```bash
kubectl get pipelineruns
kubectl logs <pipelinerun-name>-fetch-source-pod --all-containers
```

Common issues:
- **Git Authentication**: Check if the Secret is correct
- **Registry**: Check if you have push permission
- **Buildpack**: Ensure you have files like `go.mod`, `package.json` in the root

### Function Not Responding

If the function does not respond:

```bash
# View function pods
kubectl get pods -l serving.knative.dev/service=my-first-function

# View function logs
kubectl logs -l serving.knative.dev/service=my-first-function
```

Common issues:
- **Incorrect Port**: Ensure listening on the `PORT` variable
- **App not starting**: Check pod logs

### More Help

For more troubleshooting information:
- [Troubleshooting Guide](../04-reference/troubleshooting.md)

## Complete Examples

See complete examples in the repository:
- [zenith-test-functions](https://github.com/LucasGois1/zenith-test-functions) - Sample functions
- [config/samples/](https://github.com/LucasGois1/zenith-operator/tree/main/config/samples) - Function CR examples
