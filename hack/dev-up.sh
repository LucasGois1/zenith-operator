#!/bin/bash

set -e

CLUSTER_NAME="${CLUSTER_NAME:-zenith-operator-test-e2e}"
IMG="${IMG:-zenith-operator:test}"
GITHUB_USERNAME="${GITHUB_USERNAME:-LucasGois1}"

echo "🚀 Configurando ambiente de desenvolvimento..."
echo ""

# =============================================================================
# SECTION 1: Install Dependencies
# =============================================================================
echo "🔍 Verificando e instalando dependências..."

if ! command -v go &> /dev/null; then
  echo "📦 Instalando Go..."
  GO_VERSION="1.25.4"
  curl -sL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" | sudo tar -C /usr/local -xzf -
  export PATH="/usr/local/go/bin:$PATH"
  echo 'export PATH="/usr/local/go/bin:$PATH"' >> ~/.bashrc
else
  echo "✅ Go já instalado"
fi

if ! command -v kubectl &> /dev/null; then
  echo "📦 Instalando kubectl..."
  curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
  chmod +x kubectl
  sudo mv kubectl /usr/local/bin/
else
  echo "✅ kubectl já instalado"
fi

REQUIRED_KIND_VERSION="v0.24.0"
CURRENT_KIND_VERSION=$(kind version 2>/dev/null | grep -oP 'kind \K[^ ]+' || echo "none")

if [ "$CURRENT_KIND_VERSION" != "$REQUIRED_KIND_VERSION" ]; then
  echo "📦 Instalando kind ${REQUIRED_KIND_VERSION}..."
  curl -Lo ./kind "https://kind.sigs.k8s.io/dl/${REQUIRED_KIND_VERSION}/kind-linux-amd64"
  chmod +x ./kind
  sudo mv ./kind /usr/local/bin/kind
else
  echo "✅ kind ${REQUIRED_KIND_VERSION} já instalado"
fi

if ! command -v docker &> /dev/null; then
  echo "⚠️  Docker não está instalado. Por favor, instale Docker manualmente:"
  echo "   https://docs.docker.com/get-docker/"
  exit 1
else
  echo "✅ Docker já instalado"
fi

if ! command -v helm &> /dev/null; then
  echo "📦 Instalando Helm..."
  curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
else
  echo "✅ Helm já instalado"
fi

if ! command -v chainsaw &> /dev/null; then
  echo "📦 Instalando Chainsaw..."
  bash hack/install-chainsaw.sh
  export PATH="$(pwd)/bin:$PATH"
else
  echo "✅ Chainsaw já instalado"
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
    echo "📦 Iniciando registry Docker existente..."
    docker start "${REGISTRY_NAME}"
  fi
  echo "✅ Registry Docker '${REGISTRY_NAME}' já existe"
else
  # Container doesn't exist, create it
  echo "📦 Criando registry Docker local..."
  docker run -d --restart=always -p "127.0.0.1:${REGISTRY_PORT}:5000" --name "${REGISTRY_NAME}" registry:2
  echo "✅ Registry criado em localhost:${REGISTRY_PORT}"
fi

# =============================================================================
# SECTION 3: Create Kind Cluster
# =============================================================================
if ! kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  echo "📦 Criando cluster kind com configuração de registry..."
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
  echo "✅ Cluster kind '${CLUSTER_NAME}' criado"
else
  echo "✅ Cluster kind '${CLUSTER_NAME}' já existe"
fi

# Connect registry to kind network (idempotent)
if ! docker network inspect kind 2>/dev/null | grep -q "${REGISTRY_NAME}"; then
  echo "📦 Conectando registry à rede kind..."
  docker network connect kind "${REGISTRY_NAME}"
  echo "✅ Registry conectado à rede kind"
else
  echo "✅ Registry já está conectado à rede kind"
fi

# =============================================================================
# SECTION 4: Create Registry Service in Cluster
# =============================================================================
# This needs to be done before helm install because the registry service
# must exist for the operator to push images to it
if ! kubectl get namespace registry 2>/dev/null; then
  echo "📦 Criando namespace e Service para registry..."
  kubectl create namespace registry
  
  REGISTRY_IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{if eq .NetworkID "'$(docker network inspect kind -f '{{.Id}}')'"}}{{.IPAddress}}{{end}}{{end}}' "${REGISTRY_NAME}")
  
  cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: registry
  namespace: registry
spec:
  type: ClusterIP
  ports:
  - port: 5000
    targetPort: 5000
---
apiVersion: v1
kind: Endpoints
metadata:
  name: registry
  namespace: registry
subsets:
- addresses:
  - ip: ${REGISTRY_IP}
  ports:
  - port: 5000
EOF
  echo "📍 Registry acessível em: registry.registry.svc.cluster.local:5000"
  echo "📍 Registry também acessível em: localhost:${REGISTRY_PORT}"
else
  echo "✅ Registry Service já instalado"
fi

# =============================================================================
# SECTION 5: Install Platform Stack via Helm
# =============================================================================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

if ! helm list -n zenith-operator-system 2>/dev/null | grep -q "zenith-operator"; then
  echo "📦 Instalando plataforma via Helm chart (values-dev.yaml)..."
  echo "   Isso inclui: Tekton, Knative, Gateway API, MetalLB, Envoy Gateway, OpenTelemetry, Dapr"
  
  helm install zenith-operator "${PROJECT_ROOT}/charts/zenith-operator" \
    -f "${PROJECT_ROOT}/charts/zenith-operator/values-dev.yaml" \
    --namespace zenith-operator-system \
    --create-namespace \
    --wait \
    --timeout 20m
  
  echo "✅ Plataforma instalada via Helm"
else
  echo "✅ Plataforma já instalada via Helm"
  echo "   Para atualizar: helm upgrade zenith-operator ${PROJECT_ROOT}/charts/zenith-operator -f ${PROJECT_ROOT}/charts/zenith-operator/values-dev.yaml -n zenith-operator-system"
fi

# =============================================================================
# SECTION 6: Post-Helm Configuration
# =============================================================================
# Some configurations need to be done after Helm install because they depend
# on dynamically created resources (like Envoy Gateway services)

echo "📦 Verificando configuração do Knative Gateway..."

# Wait for Envoy Gateway services to be created
echo "⏳ Aguardando Envoy Gateway services serem criados..."
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
    echo "📦 Configurando Knative Gateway para usar Envoy (external + local)..."
    kubectl patch configmap/config-gateway -n knative-serving --type merge -p "{\"data\":{\"external-gateways\":\"[{\\\"class\\\":\\\"envoy\\\",\\\"gateway\\\":\\\"knative-serving/knative-gateway\\\",\\\"service\\\":\\\"envoy-gateway-system/${ENVOY_SVC}\\\"}]\",\"local-gateways\":\"[{\\\"class\\\":\\\"envoy\\\",\\\"gateway\\\":\\\"knative-serving/knative-local-gateway\\\",\\\"service\\\":\\\"envoy-gateway-system/${LOCAL_ENVOY_SVC}\\\"}]\"}}"
    echo "✅ Configurado external-gateways e local-gateways"
  else
    echo "✅ Knative Gateway já configurado para usar Envoy"
  fi
else
  echo "⚠️  Envoy Gateway services não encontrados. Verifique a instalação do Helm chart."
  echo "   Tente: kubectl get svc -n envoy-gateway-system"
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
echo "🔐 Verificando GITHUB_TOKEN..."
bash hack/verify-github-token.sh

# =============================================================================
# SECTION 9: Final Output
# =============================================================================
echo ""
echo "✅ Ambiente pronto!"
echo ""
echo "📦 Componentes instalados via Helm chart:"
echo "  - Tekton Pipelines"
echo "  - Knative Serving"
echo "  - Knative Eventing"
echo "  - Gateway API CRDs"
echo "  - Envoy Gateway"
echo "  - MetalLB (com auto-detecção de IP)"
echo "  - OpenTelemetry Operator"
echo "  - Dapr"
echo "  - Registry local"
echo ""
echo "🔍 Jaeger UI (Visualização de Traces):"
echo "  URL: http://localhost:30686"
echo "  Acesse para visualizar traces OpenTelemetry em tempo real"
echo ""
echo "Comandos úteis:"
echo "  bash hack/test-single.sh <suite>      # Executar um teste específico"
echo "  bash hack/test-debug.sh <suite>       # Executar teste com namespace preservado"
echo "  bash hack/dev-redeploy.sh             # Rebuild e redeploy rápido do operator"
echo "  bash hack/wait-pr.sh <ns> <fn>        # Aguardar PipelineRun completar"
echo "  make test-chainsaw                    # Executar todos os testes (~10 min)"
echo ""
echo "Helm commands:"
echo "  helm list -n zenith-operator-system   # Ver releases instalados"
echo "  helm upgrade zenith-operator ./charts/zenith-operator -f ./charts/zenith-operator/values-dev.yaml -n zenith-operator-system  # Atualizar"
echo "  helm uninstall zenith-operator -n zenith-operator-system  # Desinstalar"
echo ""
echo "Exemplos:"
echo "  bash hack/test-single.sh eventing-trigger"
echo "  bash hack/test-debug.sh e2e-http-basic"
echo ""
