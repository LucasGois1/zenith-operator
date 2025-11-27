#!/bin/bash


set -e

CLUSTER_NAME="${CLUSTER_NAME:-zenith-operator-test-e2e}"
IMG="${IMG:-zenith-operator:test}"

echo "🔨 Fast operator redeploy..."
echo ""

echo "1️⃣  Regenerating manifests..."
make manifests

echo ""
echo "2️⃣  Building image..."
make docker-build IMG="${IMG}"

echo ""
echo "3️⃣  Loading image into cluster..."
kind load docker-image "${IMG}" --name "${CLUSTER_NAME}"

echo ""
echo "4️⃣  Deploying operator..."
make deploy IMG="${IMG}"

echo ""
echo "5️⃣  Waiting for rollout..."
kubectl rollout status deployment/zenith-operator-controller-manager -n zenith-operator-system --timeout=2m

echo ""
echo "✅ Redeploy complete!"
echo ""
echo "📋 Controller logs (last 30s):"
kubectl logs -n zenith-operator-system deployment/zenith-operator-controller-manager --tail=50 --since=30s || true

echo ""
echo "💡 To follow logs in real-time:"
echo "   kubectl logs -n zenith-operator-system deployment/zenith-operator-controller-manager -f"
