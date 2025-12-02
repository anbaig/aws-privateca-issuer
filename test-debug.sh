#!/bin/bash
set -e

echo "=== Testing Helm Chart Setup ==="

# Check if we're in the right directory
if [ ! -f "Makefile" ]; then
    echo "Error: Not in aws-privateca-issuer directory"
    exit 1
fi

echo "1. Building manager..."
make manager

echo "2. Creating local registry..."
make create-local-registry

echo "3. Creating Kind cluster..."
make kind-cluster

echo "4. Deploying cert-manager..."
make deploy-cert-manager

echo "5. Checking cluster status..."
kubectl get nodes --kubeconfig=/tmp/pca_kubeconfig
kubectl get namespaces --kubeconfig=/tmp/pca_kubeconfig

echo "6. Running a simple helm test..."
cd tests/helm
go mod tidy
go test -v -run TestConditionalValuesCoverage -timeout=5m

echo "7. Cleanup..."
cd ../..
make kind-cluster-delete

echo "=== Test completed successfully ==="
