#!/bin/bash


set -e

CLUSTER_NAME="${CLUSTER_NAME:-zenith-operator-test-e2e}"
IMG="${IMG:-zenith-operator:test}"

echo "🔨 Fast redeploy do operator..."
echo ""

echo "1️⃣  Regenerando manifests..."
make manifests

echo ""
echo "2️⃣  Building imagem..."
make docker-build IMG="${IMG}"

echo ""
echo "3️⃣  Carregando imagem no cluster..."
kind load docker-image "${IMG}" --name "${CLUSTER_NAME}"

echo ""
echo "4️⃣  Deploying operator..."
make deploy IMG="${IMG}"

echo ""
echo "5️⃣  Aguardando rollout..."
kubectl rollout status deployment/zenith-operator-controller-manager -n zenith-operator-system --timeout=2m

echo ""
echo "✅ Redeploy completo!"
echo ""
echo "📋 Logs do controller (últimos 30s):"
kubectl logs -n zenith-operator-system deployment/zenith-operator-controller-manager --tail=50 --since=30s || true

echo ""
echo "💡 Para acompanhar logs em tempo real:"
echo "   kubectl logs -n zenith-operator-system deployment/zenith-operator-controller-manager -f"
