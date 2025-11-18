#!/bin/bash


set -e

if [ $# -lt 2 ]; then
    echo "Usage: $0 <namespace> <function-name> [--timeout 8m]"
    echo ""
    echo "Examples:"
    echo "  $0 default my-function"
    echo "  $0 chainsaw-test-ns test-function-eventing --timeout 10m"
    exit 1
fi

NAMESPACE=$1
FUNCTION_NAME=$2
TIMEOUT="${3:-8m}"

echo "🔍 Descobrindo PipelineRun para Function: $FUNCTION_NAME"

for i in $(seq 1 60); do
    PR_NAME=$(kubectl get pipelinerun -n "$NAMESPACE" -o jsonpath='{range .items[?(@.metadata.ownerReferences[0].name=="'"$FUNCTION_NAME"'")]}{.metadata.name}{"\n"}{end}' | tail -n1)
    [ -n "$PR_NAME" ] && break
    sleep 2
done

if [ -z "$PR_NAME" ]; then
    echo "❌ PipelineRun not found for Function: $FUNCTION_NAME" >&2
    exit 1
fi

echo "✅ Found PipelineRun: $PR_NAME"
echo "⏳ Aguardando conclusão (timeout: $TIMEOUT)..."

if kubectl wait --for=condition=Succeeded=True "pipelinerun/$PR_NAME" -n "$NAMESPACE" --timeout="$TIMEOUT"; then
    echo "✅ PipelineRun completed successfully"
    exit 0
else
    echo "❌ PipelineRun failed or timed out"
    echo ""
    echo "Status:"
    kubectl get pipelinerun "$PR_NAME" -n "$NAMESPACE" -o jsonpath='{.status.conditions[0]}' | jq . 2>/dev/null || kubectl get pipelinerun "$PR_NAME" -n "$NAMESPACE" -o yaml | grep -A 10 "conditions:"
    exit 1
fi
