#!/bin/bash

set -e

CLUSTER_NAME="${CLUSTER_NAME:-zenith-operator-test-e2e}"
IMG="${IMG:-zenith-operator:test}"
GITHUB_USERNAME="${GITHUB_USERNAME:-LucasGois1}"

echo "🚀 Configuring development environment..."
echo ""

# =============================================================================
# SECTION 1: Install Dependencies
# =============================================================================
echo "🔍 Checking and installing dependencies..."

if ! command -v go &> /dev/null; then
  echo "📦 Installing Go..."
  GO_VERSION="1.25.4"
  curl -sL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | sudo tar -C /usr/local -xzf -
  export PATH="/usr/local/go/bin:$PATH"
  echo 'export PATH="/usr/local/go/bin:$PATH"' >> ~/.bashrc
else
  echo "✅ Go already installed"
fi

if ! command -v kubectl &> /dev/null; then
  echo "📦 Installing kubectl..."
  curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
  chmod +x kubectl
  sudo mv kubectl /usr/local/bin/
else
  echo "✅ kubectl already installed"
fi

REQUIRED_KIND_VERSION="v0.24.0"
CURRENT_KIND_VERSION=$(kind version 2>/dev/null | grep -oP 'kind \K[^ ]+' || echo "none")

if [ "$CURRENT_KIND_VERSION" != "$REQUIRED_KIND_VERSION" ]; then
  echo "📦 Installing kind ${REQUIRED_KIND_VERSION}..."
  curl -Lo ./kind "https://kind.sigs.k8s.io/dl/${REQUIRED_KIND_VERSION}/kind-linux-amd64"
  chmod +x ./kind
  sudo mv ./kind /usr/local/bin/kind
else
  echo "✅ kind ${REQUIRED_KIND_VERSION} already installed"
fi

if ! command -v docker &> /dev/null; then
  echo "⚠️  Docker is not installed. Please install Docker manually:"
  echo "   https://docs.docker.com/get-docker/"
  exit 1
else
  echo "✅ Docker already installed"
fi

if ! command -v helm &> /dev/null; then
  echo "📦 Installing Helm..."
  curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
else
  echo "✅ Helm already installed"
fi

if ! command -v chainsaw &> /dev/null; then
  echo "📦 Installing Chainsaw..."
  bash hack/install-chainsaw.sh
  export PATH="$(pwd)/bin:$PATH"
else
  echo "✅ Chainsaw already installed"
fi

echo ""

# =============================================================================
# SECTION 2: Create Local Docker Registry
# =============================================================================
REGISTRY_NAME="kind-registry"
REGISTRY_PORT="5001"

# Ensure registry container exists (idempotent)
if docker inspect "${REGISTRY_NAME}" >/dev/null 2>&1; then
  # Container exists, check if it's running
  if [ "$(docker inspect -f '{{.State.Running}}' "${REGISTRY_NAME}")" != "true" ]; then
    echo "📦 Starting existing Docker registry..."
    docker start "${REGISTRY_NAME}"
  fi
  echo "✅ Docker registry '${REGISTRY_NAME}' already exists"
else
  # Container doesn't exist, create it
  echo "📦 Creating local Docker registry..."
  docker run -d --restart=always -p "127.0.0.1:${REGISTRY_PORT}:5000" --name "${REGISTRY_NAME}" registry:2
  echo "✅ Registry created at localhost:${REGISTRY_PORT}"
fi

# =============================================================================
# SECTION 3: Create Kind Cluster
# =============================================================================
if ! kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  echo "📦 Creating kind cluster with registry configuration..."
  cat <<EOF > /tmp/kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."registry.registry.svc.cluster.local:5000"]
    endpoint = ["http://${REGISTRY_NAME}:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.configs."registry.registry.svc.cluster.local:5000".tls]
    insecure_skip_verify = true
  [plugins."io.containerd.grpc.v1.cri".registry.configs."${REGISTRY_NAME}:5000".tls]
    insecure_skip_verify = true
EOF
  kind create cluster --name "${CLUSTER_NAME}" --image kindest/node:v1.33.0 --config /tmp/kind-config.yaml
  rm /tmp/kind-config.yaml
  echo "✅ Kind cluster '${CLUSTER_NAME}' created"
else
  echo "✅ Kind cluster '${CLUSTER_NAME}' already exists"
fi

# Connect registry to kind network (idempotent)
if ! docker network inspect kind 2>/dev/null | grep -q "${REGISTRY_NAME}"; then
  echo "📦 Connecting registry to kind network..."
  docker network connect kind "${REGISTRY_NAME}"
  echo "✅ Registry connected to kind network"
else
  echo "✅ Registry already connected to kind network"
fi

# =============================================================================
# SECTION 4: Registry Service Configuration
# =============================================================================
# The Helm chart creates the registry namespace and Service (without selector).
# We need to create the Endpoints pointing to the kind-registry container IP.
# This is done after Helm install in Section 5.1.
echo "📍 Registry will be configured by Helm chart + Endpoints"
echo "📍 External registry accessible at: localhost:${REGISTRY_PORT}"

# =============================================================================
# SECTION 5: Install Platform Stack via Helm
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

if ! helm list -n zenith-operator-system 2>/dev/null | grep -q "zenith-operator"; then
  echo "   This includes: Tekton, Knative, Gateway API, MetalLB, Envoy Gateway, OpenTelemetry, Dapr"
  
  helm install zenith-operator "${PROJECT_ROOT}/charts/zenith-operator" \
    --namespace zenith-operator-system \
    --create-namespace \
    --wait \
    --timeout 20m
  
  echo "✅ Platform installed via Helm"
else
  echo "✅ Platform already installed via Helm"
fi

# =============================================================================
# SECTION 5.1: Create Registry Endpoints for External Registry Mode
# =============================================================================
# The Helm chart creates a Service without selector in external mode.
# We need to create Endpoints pointing to the kind-registry container IP.
echo "📦 Configuring registry Endpoints..."

# Get the kind-registry container IP on the kind network
REGISTRY_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{if eq .NetworkID "'$(docker network inspect kind -f '{{.Id}}')'"}}{{.IPAddress}}{{end}}{{end}}' "${REGISTRY_NAME}")

if [ -n "$REGISTRY_IP" ]; then
  echo "📍 Registry container IP: ${REGISTRY_IP}"
  
  # Create or update the Endpoints
  cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Endpoints
metadata:
  name: registry
  namespace: registry
  labels:
    app: registry
subsets:
- addresses:
  - ip: ${REGISTRY_IP}
  ports:
  - port: 5000
    name: registry
EOF
  echo "✅ Registry Endpoints created pointing to ${REGISTRY_IP}"
else
  echo "⚠️  Could not determine registry container IP. Registry may not work correctly."
fi

# =============================================================================
# SECTION 6: Post-Helm Configuration
# =============================================================================
# Some configurations need to be done after Helm install because they depend
# on dynamically created resources (like Envoy Gateway services)

echo "📦 Checking Knative Gateway configuration..."

# Wait for Envoy Gateway services to be created
echo "⏳ Waiting for Envoy Gateway services to be created..."
for i in {1..60}; do
  ENVOY_SVC=$(kubectl get svc -n envoy-gateway-system -l gateway.envoyproxy.io/owning-gateway-name=knative-gateway -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
  LOCAL_ENVOY_SVC=$(kubectl get svc -n envoy-gateway-system -l gateway.envoyproxy.io/owning-gateway-name=knative-local-gateway -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
  
  if [ -n "$ENVOY_SVC" ] && [ -n "$LOCAL_ENVOY_SVC" ]; then
    break
  fi
  sleep 5
done

if [ -n "$ENVOY_SVC" ] && [ -n "$LOCAL_ENVOY_SVC" ]; then
  echo "📍 Envoy Gateway (external) service: envoy-gateway-system/${ENVOY_SVC}"
  echo "📍 Envoy Gateway (local) service: envoy-gateway-system/${LOCAL_ENVOY_SVC}"
  
  # Configure Knative to use Envoy Gateway services
  if ! kubectl get configmap config-gateway -n knative-serving -o yaml 2>/dev/null | grep -q "class: envoy"; then
    echo "📦 Configuring Knative Gateway to use Envoy (external + local)..."
    kubectl patch configmap/config-gateway -n knative-serving --type merge -p "{\"data\":{\"external-gateways\":\"[{\\\"class\\\":\\\"envoy\\\",\\\"gateway\\\":\\\"knative-serving/knative-gateway\\\",\\\"service\\\":\\\"envoy-gateway-system/${ENVOY_SVC}\\\"}]\",\"local-gateways\":\"[{\\\"class\\\":\\\"envoy\\\",\\\"gateway\\\":\\\"knative-serving/knative-local-gateway\\\",\\\"service\\\":\\\"envoy-gateway-system/${LOCAL_ENVOY_SVC}\\\"}]\"}}"
    echo "✅ Configured external-gateways and local-gateways"
  else
    echo "✅ Knative Gateway already configured to use Envoy"
  fi
else
  echo "⚠️  Envoy Gateway services not found. Check Helm chart installation."
  echo "   Try: kubectl get svc -n envoy-gateway-system"
fi

# =============================================================================
# SECTION 7: Build and Deploy Operator
# =============================================================================
echo ""
echo "🔨 Building operator image..."
make docker-build IMG="${IMG}"

echo "📤 Loading image into kind cluster..."
# Use docker save + ctr import as workaround for kind load issue with Kubernetes 1.33.0
# See: https://github.com/kubernetes-sigs/kind/issues/3510
if ! docker save "${IMG}" | docker exec -i "${CLUSTER_NAME}-control-plane" ctr --namespace k8s.io images import - 2>&1 | grep -q "saved"; then
  echo "⚠️  Warning: Failed to load image using ctr import, trying kind load..."
  kind load docker-image "${IMG}" --name "${CLUSTER_NAME}"
fi

echo "🚀 Deploying operator..."
make deploy IMG="${IMG}"

# =============================================================================
# SECTION 8: Verify GitHub Token
# =============================================================================
echo "🔐 Checking GITHUB_TOKEN..."
bash hack/verify-github-token.sh

# =============================================================================
# SECTION 9: Final Output
# =============================================================================
echo ""
echo "✅ Environment ready!"
echo ""
echo "📦 Components installed via Helm chart:"
echo "  - Tekton Pipelines"
echo "  - Knative Serving"
echo "  - Knative Eventing"
echo "  - Gateway API CRDs"
echo "  - Envoy Gateway"
echo "  - MetalLB (with IP auto-detection)"
echo "  - OpenTelemetry Operator"
echo "  - Dapr"
echo "  - Local Registry"
echo ""
echo "🔍 Jaeger UI (Trace Visualization):"
echo "  URL: http://localhost:30686"
echo "  Access to view OpenTelemetry traces in real-time"
echo ""
echo "Useful commands:"
echo "  bash hack/test-single.sh <suite>      # Run a specific test"
echo "  bash hack/test-debug.sh <suite>       # Run test with preserved namespace"
echo "  bash hack/dev-redeploy.sh             # Quick operator rebuild and redeploy"
echo "  bash hack/wait-pr.sh <ns> <fn>        # Wait for PipelineRun to complete"
echo "  make test-chainsaw                    # Run all tests (~10 min)"
echo ""
echo "Helm commands:"
echo "  helm list -n zenith-operator-system   # View installed releases"
echo "  helm upgrade zenith-operator ./charts/zenith-operator -n zenith-operator-system  # Upgrade"
echo "  helm uninstall zenith-operator -n zenith-operator-system  # Uninstall"
echo ""
echo "Examples:"
echo "  bash hack/test-single.sh eventing-trigger"
echo "  bash hack/test-debug.sh e2e-http-basic"
echo ""
