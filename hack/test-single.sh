#!/bin/bash


set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <test-suite-name> [chainsaw-args...]"
    echo ""
    echo "Examples:"
    echo "  $0 eventing-trigger"
    echo "  $0 eventing-trigger --skip-delete"
    echo "  $0 e2e-http-basic --timeout 15m"
    echo ""
    echo "Available test suites:"
    ls -1 test/chainsaw/ | grep -v "\.yaml$" | sed 's/^/  - /'
    exit 1
fi

SUITE=$1
shift

TEST_DIR="test/chainsaw/${SUITE}"

if [ ! -d "$TEST_DIR" ]; then
    echo "❌ Test suite not found: $SUITE"
    echo ""
    echo "Available test suites:"
    ls -1 test/chainsaw/ | grep -v "\.yaml$" | sed 's/^/  - /'
    exit 1
fi

CLUSTER_NAME="${CLUSTER_NAME:-zenith-operator-test-e2e}"

# Detect Envoy Gateway NodePort/IP
# Envoy Gateway creates services with random suffixes, so we look for the label or name pattern
ENVOY_SVC=$(kubectl get svc -n envoy-gateway-system -l gateway.envoyproxy.io/owning-gateway-name=knative-gateway -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

if [ -n "$ENVOY_SVC" ]; then
    ENVOY_NODE_PORT=$(kubectl get svc "$ENVOY_SVC" -n envoy-gateway-system -o jsonpath='{.spec.ports[?(@.name=="http")].nodePort}' 2>/dev/null || echo "N/A")
    # If LoadBalancer is used (Minikube tunnel), use the External IP
    ENVOY_EXTERNAL_IP=$(kubectl get svc "$ENVOY_SVC" -n envoy-gateway-system -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || echo "")
else
    ENVOY_NODE_PORT="N/A"
    ENVOY_EXTERNAL_IP=""
fi

NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null || echo "N/A")

echo "🧪 Executando teste: $SUITE"
echo "📍 Cluster: $CLUSTER_NAME"

if [ -n "$ENVOY_EXTERNAL_IP" ]; then
    echo "🌐 Envoy Gateway (LB): http://${ENVOY_EXTERNAL_IP}:80"
else
    echo "🌐 Envoy Gateway (NodePort): http://${NODE_IP}:${ENVOY_NODE_PORT}"
fi
echo ""

export PATH="/usr/local/go/bin:$PATH"

chainsaw test --test-dir "$TEST_DIR" "$@"
