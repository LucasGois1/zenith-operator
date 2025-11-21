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

if ! command -v kind &> /dev/null; then
  echo "📦 Instalando kind..."
  curl -Lo ./kind "https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64"
  chmod +x ./kind
  sudo mv ./kind /usr/local/bin/kind
else
  echo "✅ kind já instalado"
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

if ! kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  echo "📦 Criando cluster kind com configuração de registry..."
  cat <<EOF > /tmp/kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."registry.registry.svc.cluster.local:5000"]
    endpoint = ["http://registry.registry.svc.cluster.local:5000"]
  [plugins."io.containerd.grpc.v1.cri".registry.configs."registry.registry.svc.cluster.local:5000".tls]
    insecure_skip_verify = true
EOF
  kind create cluster --name "${CLUSTER_NAME}" --image kindest/node:v1.30.0 --config /tmp/kind-config.yaml
  rm /tmp/kind-config.yaml
else
  echo "✅ Cluster kind '${CLUSTER_NAME}' já existe"
fi

if ! kubectl get apiservices v1.tekton.dev 2>/dev/null | grep -q "v1.tekton.dev"; then
  echo "📦 Instalando Tekton Pipelines..."
  kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
  echo "⏳ Aguardando Tekton Pipelines ficar pronto..."
  kubectl wait --for=condition=ready pod -l app=tekton-pipelines-controller -n tekton-pipelines --timeout=300s
else
  echo "✅ Tekton Pipelines já instalado"
fi

if ! kubectl get apiservices v1.serving.knative.dev 2>/dev/null | grep -q "v1.serving.knative.dev"; then
  echo "📦 Instalando Knative Serving..."
  kubectl apply -f https://github.com/knative/serving/releases/latest/download/serving-crds.yaml
  kubectl apply -f https://github.com/knative/serving/releases/latest/download/serving-core.yaml
  
  echo "📦 Configurando Knative para Kubernetes 1.30.0..."
  kubectl set env deployment/controller -n knative-serving KUBERNETES_MIN_VERSION=1.30.0
  kubectl set env deployment/webhook -n knative-serving KUBERNETES_MIN_VERSION=1.30.0
  
  echo "⏳ Aguardando Knative Serving ficar pronto..."
  kubectl wait --for=condition=ready pod -l app=controller -n knative-serving --timeout=300s
  kubectl wait --for=condition=ready pod -l app=webhook -n knative-serving --timeout=300s
else
  echo "✅ Knative Serving já instalado"
fi

if ! kubectl get apiservices v1.eventing.knative.dev 2>/dev/null | grep -q "v1.eventing.knative.dev"; then
  echo "📦 Instalando Knative Eventing..."
  kubectl apply -f https://github.com/knative/eventing/releases/latest/download/eventing-crds.yaml
  kubectl apply -f https://github.com/knative/eventing/releases/latest/download/eventing-core.yaml
  echo "⏳ Aguardando Knative Eventing ficar pronto..."
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

if ! command -v helm &> /dev/null; then
  echo "📦 Instalando Helm..."
  curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
else
  echo "✅ Helm já instalado"
fi

if ! kubectl get namespace kong 2>/dev/null; then
  echo "📦 Instalando Kong Ingress Controller..."
  helm repo add kong https://charts.konghq.com
  helm repo update
  kubectl create namespace kong
  helm install kong kong/ingress -n kong \
    --set controller.ingressController.enabled=true \
    --set controller.ingressController.installCRDs=false \
    --set gateway.enabled=true \
    --set controller.ingressController.gatewayAPI.enabled=true \
    --set controller.admissionWebhook.enabled=false
  echo "⏳ Aguardando Kong ficar pronto..."
  kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=controller -n kong --timeout=300s
  kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=gateway -n kong --timeout=300s
  
  echo "📦 Configurando Kong proxy como NodePort para kind..."
  kubectl patch svc kong-gateway-proxy -n kong -p '{"spec":{"type":"NodePort"}}'
else
  echo "✅ Kong Ingress Controller já instalado"
fi

KONG_NODE_PORT=$(kubectl get svc kong-gateway-proxy -n kong -o jsonpath='{.spec.ports[?(@.name=="kong-proxy")].nodePort}')
KONG_NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}')
if [ -n "$KONG_NODE_PORT" ] && [ -n "$KONG_NODE_IP" ]; then
  echo "📍 Kong proxy acessível em: http://${KONG_NODE_IP}:${KONG_NODE_PORT}"
  echo "   Use com Host header para acessar functions: curl -H 'Host: <function-url>' http://${KONG_NODE_IP}:${KONG_NODE_PORT}"
fi

if ! kubectl get namespace registry 2>/dev/null; then
  echo "📦 Instalando registry local para buildpacks..."
  kubectl create namespace registry
  cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: registry-pvc
  namespace: registry
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: registry
  namespace: registry
spec:
  replicas: 1
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
        ports:
        - containerPort: 5000
        volumeMounts:
        - name: registry-storage
          mountPath: /var/lib/registry
      volumes:
      - name: registry-storage
        persistentVolumeClaim:
          claimName: registry-pvc
---
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
  selector:
    app: registry
EOF
  echo "⏳ Aguardando registry ficar pronto..."
  kubectl wait --for=condition=available --timeout=60s deployment/registry -n registry
  echo "📍 Registry local acessível em: registry.registry.svc.cluster.local:5000"
else
  echo "✅ Registry local já instalado"
fi

if ! kubectl get deployment net-gateway-api-controller -n knative-serving 2>/dev/null; then
  echo "📦 Instalando Knative net-gateway-api..."
  kubectl apply -f https://github.com/knative-extensions/net-gateway-api/releases/download/knative-v1.17.0/net-gateway-api.yaml
  echo "⏳ Aguardando net-gateway-api ficar pronto..."
  kubectl wait --for=condition=ready pod -l app=net-gateway-api-controller -n knative-serving --timeout=300s
else
  echo "✅ Knative net-gateway-api já instalado"
fi

if ! kubectl get configmap config-network -n knative-serving -o yaml | grep -q "ingress-class: gateway-api.ingress.networking.knative.dev"; then
  echo "📦 Configurando Knative para usar Gateway API..."
  kubectl patch configmap/config-network -n knative-serving --type merge -p '{"data":{"ingress-class":"gateway-api.ingress.networking.knative.dev"}}'
fi

if ! kubectl get gatewayclass kong 2>/dev/null; then
  echo "📦 Criando GatewayClass e Gateway para Kong..."
  cat <<EOF | kubectl apply -f -
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: kong
  annotations:
    konghq.com/gatewayclass-unmanaged: "true"
spec:
  controllerName: konghq.com/kic-gateway-controller
---
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: knative-gateway
  namespace: knative-serving
spec:
  gatewayClassName: kong
  listeners:
  - name: http
    protocol: HTTP
    port: 80
    allowedRoutes:
      namespaces:
        from: All
EOF
  echo "⏳ Aguardando Gateway ficar pronto..."
  sleep 5
fi

if ! kubectl get configmap config-gateway -n knative-serving -o yaml | grep -q "class: kong"; then
  echo "📦 Configurando Knative Gateway para usar Kong..."
  kubectl patch configmap/config-gateway -n knative-serving --type merge -p '{"data":{"local-gateways":"- class: kong\n  gateway: knative-serving/knative-gateway\n  service: kong/kong-gateway-proxy\n  supported-features:\n  - HTTPRouteRequestTimeout\n"}}'
fi

echo "🔨 Building operator image..."
make docker-build IMG="${IMG}"

echo "📤 Loading image into kind cluster..."
kind load docker-image "${IMG}" --name "${CLUSTER_NAME}"

echo "🚀 Deploying operator..."
make deploy IMG="${IMG}"

echo "🔐 Verificando GITHUB_TOKEN..."
bash hack/verify-github-token.sh

echo ""
echo "✅ Ambiente pronto!"
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
