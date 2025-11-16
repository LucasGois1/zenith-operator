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
