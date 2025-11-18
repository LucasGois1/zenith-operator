#!/bin/bash

set -e

CLUSTER_NAME="${CLUSTER_NAME:-zenith-operator-test-e2e}"
IMG="${IMG:-zenith-operator:test}"
GITHUB_USERNAME="${GITHUB_USERNAME:-LucasGois1}"

echo "🚀 Configurando ambiente de desenvolvimento..."

if ! kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
  echo "📦 Criando cluster kind..."
  kind create cluster --name "${CLUSTER_NAME}"
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
  echo "⏳ Aguardando Knative Serving ficar pronto..."
  kubectl wait --for=condition=ready pod -l app=controller -n knative-serving --timeout=300s
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
else
  echo "✅ Kong Ingress Controller já instalado"
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
echo "  make test-chainsaw                    # Executar todos os testes (~10 min)"
echo "  make test-chainsaw-git                # Executar apenas teste de git-clone (~2 min)"
echo "  make test-chainsaw-basic              # Executar apenas teste básico (~10 min)"
echo "  make dev-redeploy                     # Rebuild e redeploy rápido"
echo ""
