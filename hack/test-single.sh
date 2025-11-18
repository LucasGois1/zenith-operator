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
KONG_NODE_PORT=$(kubectl get svc kong-gateway-proxy -n kong -o jsonpath='{.spec.ports[?(@.name=="kong-proxy")].nodePort}' 2>/dev/null || echo "N/A")
KONG_NODE_IP=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null || echo "N/A")

echo "🧪 Executando teste: $SUITE"
echo "📍 Cluster: $CLUSTER_NAME"
echo "🌐 Kong: http://${KONG_NODE_IP}:${KONG_NODE_PORT}"
echo ""

export PATH="/usr/local/go/bin:$PATH"

chainsaw test --test-dir "$TEST_DIR" "$@"
