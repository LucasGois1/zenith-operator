#!/bin/bash

set -e

CLUSTER_NAME="${CLUSTER_NAME:-zenith-operator-test-e2e}"
IMG="${IMG:-zenith-operator:test}"
GITHUB_USERNAME="${GITHUB_USERNAME:-LucasGois1}"

echo "🚀 Configurando ambiente de desenvolvimento..."
echo ""

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

if ! command -v chainsaw &> /dev/null; then
  echo "📦 Instalando Chainsaw..."
  bash hack/install-chainsaw.sh
  export PATH="$(pwd)/bin:$PATH"
else
  echo "✅ Chainsaw já instalado"
fi

echo ""

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

# Create kind cluster if it doesn't exist
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

if ! kubectl get apiservices v1.tekton.dev 2>/dev/null | grep -q "v1.tekton.dev"; then
  echo "📦 Instalando Tekton Pipelines..."
  kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
  echo "⏳ Aguardando Tekton Pipelines ficar pronto..."
  # Wait for pods to be created before waiting for them to be ready
  for i in {1..30}; do
    if kubectl get pod -l app=tekton-pipelines-controller -n tekton-pipelines 2>/dev/null | grep -q tekton; then
      break
    fi
    sleep 2
  done
  kubectl wait --for=condition=ready pod -l app=tekton-pipelines-controller -n tekton-pipelines --timeout=300s
  
  # Enable step-actions feature flag required for buildpacks-phases Task
  # This Task uses Step Results and Step When expressions
  echo "📦 Configurando Tekton feature flags..."
  kubectl patch configmap feature-flags -n tekton-pipelines --type merge -p '{"data":{"enable-step-actions":"true"}}'
else
  echo "✅ Tekton Pipelines já instalado"
  # Ensure feature flag is set even if Tekton was already installed
  kubectl patch configmap feature-flags -n tekton-pipelines --type merge -p '{"data":{"enable-step-actions":"true"}}' 2>/dev/null || true
fi

if ! kubectl get apiservices v1.serving.knative.dev 2>/dev/null | grep -q "v1.serving.knative.dev"; then
  echo "📦 Instalando Knative Serving..."
  kubectl apply -f https://github.com/knative/serving/releases/latest/download/serving-crds.yaml
  kubectl apply -f https://github.com/knative/serving/releases/latest/download/serving-core.yaml
  
  echo "📦 Configurando Knative para Kubernetes 1.33.0..."
  kubectl set env deployment/controller -n knative-serving KUBERNETES_MIN_VERSION=1.33.0
  kubectl set env deployment/webhook -n knative-serving KUBERNETES_MIN_VERSION=1.33.0
  kubectl set env deployment/activator -n knative-serving KUBERNETES_MIN_VERSION=1.33.0
  kubectl set env deployment/autoscaler -n knative-serving KUBERNETES_MIN_VERSION=1.33.0
  
  echo "⏳ Aguardando Knative Serving ficar pronto..."
  # Wait for pods to be created before waiting for them to be ready
  for i in {1..30}; do
    if kubectl get pod -l app=controller -n knative-serving 2>/dev/null | grep -q controller; then
      break
    fi
    sleep 2
  done
  kubectl wait --for=condition=ready pod -l app=controller -n knative-serving --timeout=300s
  kubectl wait --for=condition=ready pod -l app=autoscaler -n knative-serving --timeout=300s
  kubectl wait --for=condition=ready pod -l app=activator -n knative-serving --timeout=300s
  kubectl wait --for=condition=ready pod -l app=webhook -n knative-serving --timeout=300s
else
  echo "✅ Knative Serving já instalado"
fi

if ! kubectl get apiservices v1.eventing.knative.dev 2>/dev/null | grep -q "v1.eventing.knative.dev"; then
  echo "📦 Instalando Knative Eventing..."
  kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.20.0/eventing-crds.yaml
  kubectl apply -f https://github.com/knative/eventing/releases/download/knative-v1.20.0/eventing-core.yaml
  echo "⏳ Aguardando Knative Eventing ficar pronto..."
  # Wait for pods to be created before waiting for them to be ready
  for i in {1..30}; do
    if kubectl get pod -l app=eventing-controller -n knative-eventing 2>/dev/null | grep -q eventing; then
      break
    fi
    sleep 2
  done
  kubectl wait --for=condition=ready pod -l app=eventing-controller -n knative-eventing --timeout=300s
else
  echo "✅ Knative Eventing já instalado"
fi

if ! kubectl get crd gateways.gateway.networking.k8s.io 2>/dev/null; then
  echo "📦 Instalando Gateway API CRDs..."
  kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml
else
  echo "✅ Gateway API CRDs já instalados"
fi

if ! kubectl get namespace metallb-system 2>/dev/null; then
  echo "📦 Instalando MetalLB para LoadBalancer support em kind..."
  kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.14.9/config/manifests/metallb-native.yaml
  
  echo "⏳ Aguardando MetalLB ficar pronto..."
  # Wait for pods to be created before waiting for them to be ready
  for i in {1..30}; do
    if kubectl get pod -l app=metallb -n metallb-system 2>/dev/null | grep -q metallb; then
      break
    fi
    sleep 2
  done
  kubectl wait --for=condition=ready pod -l app=metallb -n metallb-system --timeout=120s
  
  echo "📦 Configurando MetalLB IP address pool..."
  # Extract only the IPv4 subnet (contains dots, not colons)
  DOCKER_NETWORK=$(docker network inspect kind -f '{{range .IPAM.Config}}{{.Subnet}}{{"\n"}}{{end}}' | grep '\.' | head -n1)
  # Get the network base IP (e.g., 172.19.0.0/24 -> 172.19.0.0)
  NETWORK_IP=$(echo ${DOCKER_NETWORK} | cut -d'/' -f1)
  # Get the first three octets to stay within the subnet (e.g., 172.19.0)
  IP_PREFIX=$(echo ${NETWORK_IP} | cut -d'.' -f1-3)
  
  cat <<EOF | kubectl apply -f -
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: kind-pool
  namespace: metallb-system
spec:
  addresses:
  - ${IP_PREFIX}.200-${IP_PREFIX}.250
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: kind-l2
  namespace: metallb-system
spec:
  ipAddressPools:
  - kind-pool
EOF
  
  echo "✅ MetalLB configurado com IP pool ${IP_PREFIX}.200-${IP_PREFIX}.250"
else
  echo "✅ MetalLB já instalado"
fi

if ! command -v helm &> /dev/null; then
  echo "📦 Instalando Helm..."
  curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
else
  echo "✅ Helm já instalado"
fi

if ! kubectl get namespace envoy-gateway-system 2>/dev/null; then
  echo "📦 Instalando Envoy Gateway..."
  
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
  
  curl -sL https://github.com/envoyproxy/gateway/releases/download/v1.6.0/install.yaml > /tmp/envoy-gateway-install.yaml
  cd /tmp && csplit -s -f envoy-gateway- envoy-gateway-install.yaml '/^---$/' '{*}'
  
  for file in envoy-gateway-*; do
    if ! grep -q "kind: CustomResourceDefinition" "$file"; then
      kubectl apply -f "$file" 2>&1 | grep -v "unchanged" || true
    fi
  done
  
  cd "${PROJECT_ROOT}"
  
  echo "⏳ Aguardando Envoy Gateway ficar pronto..."
  # Wait for deployment to be created before waiting for it to be available
  for i in {1..30}; do
    if kubectl get deployment envoy-gateway -n envoy-gateway-system 2>/dev/null | grep -q envoy-gateway; then
      break
    fi
    sleep 2
  done
  kubectl wait --for=condition=available --timeout=300s deployment/envoy-gateway -n envoy-gateway-system
else
  echo "✅ Envoy Gateway já instalado"
fi

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

if ! kubectl get deployment net-gateway-api-controller -n knative-serving 2>/dev/null; then
  echo "📦 Instalando Knative net-gateway-api..."
  kubectl apply -f https://github.com/knative-extensions/net-gateway-api/releases/download/knative-v1.20.0/net-gateway-api.yaml
  
  echo "⏳ Aguardando net-gateway-api ficar pronto..."
  # Wait for pods to be created before waiting for them to be ready
  for i in {1..30}; do
    if kubectl get pod -l app=net-gateway-api-controller -n knative-serving 2>/dev/null | grep -q net-gateway; then
      break
    fi
    sleep 2
  done
  kubectl wait --for=condition=ready pod -l app=net-gateway-api-controller -n knative-serving --timeout=300s
else
  echo "✅ Knative net-gateway-api já instalado"
fi

if ! kubectl get configmap config-network -n knative-serving -o yaml | grep -q "ingress-class: gateway-api.ingress.networking.knative.dev"; then
  echo "📦 Configurando Knative para usar Gateway API..."
  kubectl patch configmap/config-network -n knative-serving --type merge -p '{"data":{"ingress-class":"gateway-api.ingress.networking.knative.dev"}}'
fi

if ! kubectl get gatewayclass envoy 2>/dev/null; then
  echo "📦 Criando GatewayClass e Gateways para Envoy..."
  cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: envoy
spec:
  controllerName: gateway.envoyproxy.io/gatewayclass-controller
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: knative-gateway
  namespace: knative-serving
spec:
  gatewayClassName: envoy
  listeners:
  - name: http
    protocol: HTTP
    port: 80
    allowedRoutes:
      namespaces:
        from: All
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: knative-local-gateway
  namespace: knative-serving
spec:
  gatewayClassName: envoy
  listeners:
  - name: http
    protocol: HTTP
    port: 80
    allowedRoutes:
      namespaces:
        from: All
EOF
  echo "⏳ Aguardando Gateways ficarem prontos..."
  sleep 15
  
  ENVOY_SVC=$(kubectl get svc -n envoy-gateway-system -l gateway.envoyproxy.io/owning-gateway-name=knative-gateway -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
  LOCAL_ENVOY_SVC=$(kubectl get svc -n envoy-gateway-system -l gateway.envoyproxy.io/owning-gateway-name=knative-local-gateway -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
  
  if [ -n "$ENVOY_SVC" ]; then
    echo "📍 Envoy Gateway (external) service: envoy-gateway-system/${ENVOY_SVC}"
  fi
  if [ -n "$LOCAL_ENVOY_SVC" ]; then
    echo "📍 Envoy Gateway (local) service: envoy-gateway-system/${LOCAL_ENVOY_SVC}"
  fi
fi

if ! kubectl get configmap config-gateway -n knative-serving -o yaml | grep -q "class: envoy"; then
  echo "📦 Configurando Knative Gateway para usar Envoy (external + local)..."
  
  ENVOY_SVC=$(kubectl get svc -n envoy-gateway-system -l gateway.envoyproxy.io/owning-gateway-name=knative-gateway -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
  LOCAL_ENVOY_SVC=$(kubectl get svc -n envoy-gateway-system -l gateway.envoyproxy.io/owning-gateway-name=knative-local-gateway -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
  
  if [ -n "$ENVOY_SVC" ] && [ -n "$LOCAL_ENVOY_SVC" ]; then
    kubectl patch configmap/config-gateway -n knative-serving --type merge -p "{\"data\":{\"external-gateways\":\"[{\\\"class\\\":\\\"envoy\\\",\\\"gateway\\\":\\\"knative-serving/knative-gateway\\\",\\\"service\\\":\\\"envoy-gateway-system/${ENVOY_SVC}\\\"}]\",\"local-gateways\":\"[{\\\"class\\\":\\\"envoy\\\",\\\"gateway\\\":\\\"knative-serving/knative-local-gateway\\\",\\\"service\\\":\\\"envoy-gateway-system/${LOCAL_ENVOY_SVC}\\\"}]\"}}"
    echo "✅ Configurado external-gateways e local-gateways"
  else
    echo "⚠️  Envoy Gateway services not found yet, will be configured on first reconciliation"
  fi
fi

if ! kubectl get namespace cert-manager 2>/dev/null; then
  echo "📦 Instalando cert-manager (prerequisite for OpenTelemetry Operator)..."
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.2/cert-manager.yaml
  
  echo "⏳ Aguardando cert-manager ficar pronto..."
  # Wait for pods to be created before waiting for them to be ready
  for i in {1..30}; do
    if kubectl get pod -l app=cert-manager -n cert-manager 2>/dev/null | grep -q cert-manager; then
      break
    fi
    sleep 2
  done
  kubectl wait --for=condition=ready pod -l app=cert-manager -n cert-manager --timeout=300s
  kubectl wait --for=condition=ready pod -l app=webhook -n cert-manager --timeout=300s
  kubectl wait --for=condition=ready pod -l app=cainjector -n cert-manager --timeout=300s
else
  echo "✅ cert-manager já instalado"
fi

if ! kubectl get namespace opentelemetry-operator-system 2>/dev/null; then
  echo "📦 Instalando OpenTelemetry Operator..."
  
  # Add OpenTelemetry Helm repository
  helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts 2>/dev/null || true
  helm repo update
  
  # Install OpenTelemetry Operator
  helm install opentelemetry-operator open-telemetry/opentelemetry-operator \
    --namespace opentelemetry-operator-system \
    --create-namespace \
    --version 0.99.2 \
    --set "manager.collectorImage.repository=otel/opentelemetry-collector-k8s" \
    --wait --timeout=300s
  
  echo "⏳ Aguardando OpenTelemetry Operator ficar pronto..."
  kubectl wait --for=condition=available --timeout=300s deployment/opentelemetry-operator -n opentelemetry-operator-system
  
  echo "📦 Criando namespace observability para Jaeger..."
  kubectl create namespace observability 2>/dev/null || true
  
  echo "📦 Instalando Jaeger all-in-one..."
  cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jaeger
  namespace: observability
  labels:
    app: jaeger
spec:
  replicas: 1
  selector:
    matchLabels:
      app: jaeger
  template:
    metadata:
      labels:
        app: jaeger
    spec:
      containers:
      - name: jaeger
        image: jaegertracing/all-in-one:latest
        env:
        - name: COLLECTOR_OTLP_ENABLED
          value: "true"
        ports:
        - containerPort: 16686
          name: ui
        - containerPort: 4317
          name: otlp-grpc
        - containerPort: 4318
          name: otlp-http
        - containerPort: 14250
          name: model-proto
        resources:
          limits:
            memory: 512Mi
          requests:
            memory: 256Mi
---
apiVersion: v1
kind: Service
metadata:
  name: jaeger
  namespace: observability
  labels:
    app: jaeger
spec:
  type: ClusterIP
  ports:
  - port: 16686
    targetPort: ui
    name: ui
  - port: 4317
    targetPort: otlp-grpc
    name: otlp-grpc
  - port: 4318
    targetPort: otlp-http
    name: otlp-http
  - port: 14250
    targetPort: model-proto
    name: model-proto
  selector:
    app: jaeger
---
apiVersion: v1
kind: Service
metadata:
  name: jaeger-ui
  namespace: observability
  labels:
    app: jaeger
spec:
  type: NodePort
  ports:
  - port: 16686
    targetPort: ui
    nodePort: 30686
    name: ui
  selector:
    app: jaeger
EOF
  
  echo "⏳ Aguardando Jaeger ficar pronto..."
  kubectl wait --for=condition=available --timeout=300s deployment/jaeger -n observability
  
  echo "📦 Criando OpenTelemetry Collector com exporters para debug e Jaeger..."
  cat <<EOF | kubectl apply -f -
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: otel-collector
  namespace: opentelemetry-operator-system
spec:
  mode: deployment
  config:
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318
    processors:
      batch: {}
    exporters:
      debug:
        verbosity: detailed
      otlp/jaeger:
        endpoint: jaeger.observability.svc.cluster.local:4317
        tls:
          insecure: true
    service:
      pipelines:
        traces:
          receivers: [otlp]
          processors: [batch]
          exporters: [debug, otlp/jaeger]
EOF
  
  echo "⏳ Aguardando OpenTelemetry Collector ficar pronto..."
  sleep 10
  kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=opentelemetry-collector -n opentelemetry-operator-system --timeout=300s
  
  echo "✅ OpenTelemetry Operator, Collector e Jaeger instalados"
else
  echo "✅ OpenTelemetry Operator já instalado"
fi

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

echo "🔐 Verificando GITHUB_TOKEN..."
bash hack/verify-github-token.sh

echo ""
echo "✅ Ambiente pronto!"
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
echo "Exemplos:"
echo "  bash hack/test-single.sh eventing-trigger"
echo "  bash hack/test-debug.sh e2e-http-basic"
echo ""
