# Creating Synchronous HTTP Functions

This guide shows how to create functions that respond to synchronous HTTP requests using Zenith Operator.

## Overview

Synchronous HTTP functions are ideal for:
- REST APIs
- Webhooks
- Microservices
- HTTP endpoints that return immediate responses

Zenith Operator automatically:
1. Clones your Git repository
2. Builds a container image using Buildpacks
3. Deploys as a Knative Service
4. Exposes a public URL accessible via HTTP

## Prerequisites

- Kubernetes cluster with Zenith Operator installed
- Git repository with function code
- Container registry (or use local registry)
- Git authentication Secret (if private repository)

## Function Code Structure

Your function must be an HTTP application listening on a port (usually 8080). Zenith Operator uses Cloud Native Buildpacks to automatically detect the language and build the image.

### Go Example

```go
package main

import (
    "encoding/json"
    "log"
    "net/http"
    "os"
)

type Response struct {
    Status  string `json:"status"`
    Message string `json:"message"`
    Type    string `json:"type"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    response := Response{
        Status:  "ok",
        Message: "Hello from Zenith Function!",
        Type:    "http-sync",
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }
    
    http.HandleFunc("/", handler)
    log.Printf("Listening on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

### Python Example

```python
from flask import Flask, jsonify
import os

app = Flask(__name__)

@app.route('/')
def hello():
    return jsonify({
        'status': 'ok',
        'message': 'Hello from Zenith Function!',
        'type': 'http-sync'
    })

if __name__ == '__main__':
    port = int(os.environ.get('PORT', 8080))
    app.run(host='0.0.0.0', port=port)
```

### Node.js Example

```javascript
const express = require('express');
const app = express();

app.get('/', (req, res) => {
    res.json({
        status: 'ok',
        message: 'Hello from Zenith Function!',
        type: 'http-sync'
    });
});

const port = process.env.PORT || 8080;
app.listen(port, () => {
    console.log(`Listening on port ${port}`);
});
```

## Step 1: Prepare Git Repository

1. Create a Git repository with your function code
2. Ensure the code is at the root of the repository
3. Commit and push to GitHub/GitLab

```bash
git init
git add .
git commit -m "Initial function implementation"
git remote add origin https://github.com/myorg/my-function
git push -u origin main
```

## Step 2: Create Git Authentication Secret (Optional)

If your repository is private, create a Secret for authentication:

```bash
# Create secret with credentials
kubectl create secret generic github-auth \
  --from-literal=username=myusername \
  --from-literal=password=ghp_mytoken \
  --type=kubernetes.io/basic-auth

# Add annotation for Tekton
kubectl annotate secret github-auth \
  tekton.dev/git-0=https://github.com
```

**Note**: For GitHub, use a Personal Access Token (PAT) as password.

## Step 3: Create Function Custom Resource

Create a YAML file with the function definition:

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: my-http-function
  namespace: default
spec:
  # Git repository with code
  gitRepo: https://github.com/myorg/my-function
  gitRevision: main
  
  # Authentication secret (optional)
  gitAuthSecretName: github-auth
  
  # Build configuration
  build:
    # Target image
    image: registry.example.com/my-http-function:latest
    
    # Registry secret (optional)
    # registrySecretName: registry-credentials
  
  # Deploy configuration
  deploy: {}
```

Apply the resource:

```bash
kubectl apply -f my-http-function.yaml
```

## Step 4: Monitor Build

The operator will automatically create a Tekton PipelineRun to build the image:

```bash
# Check function status
kubectl get functions

# Check function details
kubectl describe function my-http-function

# Check PipelineRuns
kubectl get pipelineruns

# Check build logs
kubectl logs -f <pipelinerun-name>-fetch-source-pod --all-containers
```

The function status will go through these phases:
1. **Building**: Build in progress
2. **BuildSucceeded**: Build completed successfully
3. **Ready**: Function deployed and ready to receive requests

## Step 5: Access the Function

After deployment, the function will be accessible via URL:

```bash
# Get function URL
FUNCTION_URL=$(kubectl get function my-http-function -o jsonpath='{.status.url}')
echo "Function URL: $FUNCTION_URL"

# Make a request
curl $FUNCTION_URL

# Expected response:
# {"status":"ok","message":"Hello from Zenith Function!","type":"http-sync"}
```

### Access via Envoy Gateway

If accessing from outside the cluster, use Envoy Gateway:

```bash
# Get Envoy Gateway endpoint
ENVOY_IP=$(kubectl get svc -n envoy-gateway-system \
  -l gateway.envoyproxy.io/owning-gateway-name=knative-gateway \
  -o jsonpath='{.items[0].status.loadBalancer.ingress[0].ip}')

# Get function hostname
FUNCTION_HOST=$(echo $FUNCTION_URL | sed 's|http://||' | sed 's|https://||')

# Make request with Host header
curl -H "Host: $FUNCTION_HOST" http://$ENVOY_IP/
```

## Step 6: Update the Function

To update the function, make changes to the code and push to Git:

```bash
# Make code changes
git add .
git commit -m "Update function"
git push

# Update Function CR to trigger rebuild
kubectl annotate function my-http-function \
  rebuild=$(date +%s) --overwrite
```

Or update the Git revision in spec:

```yaml
spec:
  gitRevision: v2.0.0  # New tag or branch
```

## Advanced Configurations

### Environment Variables

Add environment variables to your function:

```yaml
spec:
  deploy:
    env:
      - name: DATABASE_URL
        value: postgres://db.example.com/mydb
      - name: API_KEY
        value: secret-key
```

### Dapr Integration

Enable Dapr sidecar for service mesh:

```yaml
spec:
  deploy:
    dapr:
      enabled: true
      appID: my-http-function
      appPort: 8080
```

With Dapr enabled, you can use:
- Service discovery
- Pub/Sub
- State management
- Secret stores

### Optimizing Performance with Scale Configuration

For critical APIs needing low latency, configure autoscaling behavior to eliminate cold starts.

#### Eliminating Cold Starts

```yaml
apiVersion: functions.zenith.com/v1alpha1
kind: Function
metadata:
  name: critical-api
spec:
  gitRepo: https://github.com/myorg/critical-api
  gitRevision: main
  build:
    image: registry.example.com/critical-api:latest
  deploy:
    scale:
      minScale: 1      # Always keep 1 replica active
      maxScale: 20     # Limit scalability to control costs
    env:
      - name: LOG_LEVEL
        value: info
```

#### Cost vs Performance Considerations

- **minScale: 0** (default) - Scale-to-zero, maximum savings, cold start present
- **minScale: 1** - No cold start, cost of 1 pod always active
- **maxScale** - Controls scalability ceiling and costs during peaks

#### Recommendations by Environment

**Development:**
```yaml
deploy:
  scale: {}  # Use defaults (scale-to-zero)
```

**Production - Critical API:**
```yaml
deploy:
  scale:
    minScale: 2    # Redundancy
    maxScale: 100  # High traffic support
```

**Production - Non-critical API:**
```yaml
deploy:
  scale:
    maxScale: 10   # Cost control only
```

### Private Registry

To use a private registry, create a Secret:

```bash
kubectl create secret docker-registry registry-credentials \
  --docker-server=registry.example.com \
  --docker-username=myuser \
  --docker-password=mypassword \
  --docker-email=myemail@example.com
```

And reference it in the Function:

```yaml
spec:
  build:
    image: registry.example.com/my-http-function:latest
    registrySecretName: registry-credentials
```

## Troubleshooting

### Build Failed

If build fails, check logs:

```bash
# Check PipelineRuns
kubectl get pipelineruns

# Check PipelineRun logs
kubectl describe pipelinerun <pipelinerun-name>

# Check detailed logs
kubectl logs <pipelinerun-name>-fetch-source-pod --all-containers
```

Common issues:
- **Git Authentication failed**: Check Secret and token
- **Buildpack didn't detect language**: Ensure files like `go.mod`, `package.json`, `requirements.txt` exist at root
- **Registry push failed**: Check registry credentials

### Function Not Responding

If function doesn't respond:

```bash
# Check Knative Service status
kubectl get ksvc

# Check function pods
kubectl get pods -l serving.knative.dev/service=my-http-function

# Check function logs
kubectl logs -l serving.knative.dev/service=my-http-function
```

Common issues:
- **Incorrect Port**: Ensure listening on port specified by `PORT` variable
- **App not starting**: Check pod logs
- **Timeout**: Function takes too long to respond (scale-from-zero)

### URL Not Accessible

If URL is not accessible:

```bash
# Check Envoy Gateway
kubectl get svc -n envoy-gateway-system

# Check HTTPRoute
kubectl get httproute

# Check Gateway
kubectl get gateway -n knative-serving
```

## Complete Examples

See complete examples in the repository:
- [zenith-test-functions](https://github.com/LucasGois1/zenith-test-functions) - Basic Go function
- [config/samples/](https://github.com/LucasGois1/zenith-operator/tree/main/config/samples) - Function CR examples

## Next Steps

- [Creating Asynchronous Event Functions](event-functions.md)
- [Function Communication](function-communication.md)
- [Function CRD Specification](../04-reference/function-crd.md)
- [Operator Reference](../04-reference/operator-reference.md)
