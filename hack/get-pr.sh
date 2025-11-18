#!/bin/bash


set -e

if [ $# -lt 2 ]; then
    echo "Usage: $0 <namespace> <function-name>"
    echo ""
    echo "Examples:"
    echo "  $0 default my-function"
    echo "  $0 chainsaw-test-ns test-function-eventing"
    exit 1
fi

NAMESPACE=$1
FUNCTION_NAME=$2

PR_NAME=$(kubectl get pipelinerun -n "$NAMESPACE" -o jsonpath='{range .items[?(@.metadata.ownerReferences[0].name=="'"$FUNCTION_NAME"'")]}{.metadata.name}{"\n"}{end}' | tail -n1)

if [ -z "$PR_NAME" ]; then
    echo "❌ PipelineRun not found for Function: $FUNCTION_NAME in namespace: $NAMESPACE" >&2
    exit 1
fi

echo "$PR_NAME"
