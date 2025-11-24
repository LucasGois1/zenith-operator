#!/bin/bash


set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <test-suite-name>"
    echo ""
    echo "This script runs a test with --skip-delete and automatically collects:"
    echo "  - PipelineRun and TaskRun status"
    echo "  - Pod logs (git-clone, buildpacks phases)"
    echo "  - Operator controller logs"
    echo ""
    echo "Examples:"
    echo "  $0 eventing-trigger"
    echo "  $0 e2e-http-basic"
    exit 1
fi

SUITE=$1
TEST_DIR="test/chainsaw/${SUITE}"

if [ ! -d "$TEST_DIR" ]; then
    echo "❌ Test suite not found: $SUITE"
    exit 1
fi

echo "🐛 Executando teste em modo debug: $SUITE"
echo "📍 Namespace será preservado para inspeção"
echo ""

export PATH="/usr/local/go/bin:$PATH"

chainsaw test --test-dir "$TEST_DIR" --skip-delete || true

echo ""
echo "🔍 Coletando informações de debug..."
echo ""

NAMESPACE=$(kubectl get namespaces -o name | grep "chainsaw-" | head -1 | cut -d/ -f2)

if [ -z "$NAMESPACE" ]; then
    echo "⚠️  Nenhum namespace chainsaw-* encontrado"
    exit 0
fi

echo "📦 Namespace: $NAMESPACE"
echo ""

echo "📋 Recursos no namespace:"
kubectl get all,pipelinerun,taskrun,function -n "$NAMESPACE"
echo ""

PR_NAME=$(kubectl get pipelinerun -n "$NAMESPACE" -o name | head -1 | cut -d/ -f2)
if [ -n "$PR_NAME" ]; then
    echo "🔧 PipelineRun: $PR_NAME"
    kubectl get pipelinerun "$PR_NAME" -n "$NAMESPACE" -o jsonpath='{.status.conditions[0]}' | jq . 2>/dev/null || kubectl get pipelinerun "$PR_NAME" -n "$NAMESPACE" -o yaml | grep -A 10 "conditions:"
    echo ""
    
    echo "📝 TaskRun logs:"
    for TR in $(kubectl get taskrun -n "$NAMESPACE" -o name); do
        TR_NAME=$(echo "$TR" | cut -d/ -f2)
        echo ""
        echo "=== $TR_NAME ==="
        kubectl logs -n "$NAMESPACE" "$TR_NAME" --all-containers --tail=50 2>/dev/null || echo "  (no logs available)"
    done
fi

echo ""
echo "🎛️  Operator logs (últimos 2 minutos):"
kubectl logs -n zenith-operator-system deployment/zenith-operator-controller-manager --tail=100 --since=2m || true

echo ""
echo "💡 Para limpar o namespace:"
echo "   kubectl delete namespace $NAMESPACE"
