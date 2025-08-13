#!/bin/bash
set -euo pipefail

readonly CLUSTER_NAME="gocommerce"
readonly KIND_CONFIG="$(dirname "$0")/kind-config.yaml"

install_operator() {
    echo "🔎 Checking for CloudNativePG operator..."
    if helm status cnpg -n cnpg-system > /dev/null 2>&1; then
        echo "✅ CloudNativePG operator is already installed."
    else
        echo "Adding cnpg chart repository..."
        helm repo add cnpg https://cloudnative-pg.github.io/charts || true
        helm repo update cnpg
        echo "📦 Installing CloudNativePG operator..."
        helm install cnpg cnpg/cloudnative-pg --create-namespace --namespace cnpg-system --wait
        echo "✅ CloudNativePG operator installed successfully."
    fi
}

install_ingress_controller() {
    echo "📦 Installing Nginx Ingress controller for Kind..."
    kubectl apply -f https://kind.sigs.k8s.io/examples/ingress/deploy-ingress-nginx.yaml
    echo "⌛ Waiting for Ingress controller to be ready..."
    kubectl wait --namespace ingress-nginx \
      --for=condition=ready pod \
      --selector=app.kubernetes.io/component=controller \
      --timeout=180s
    echo "✅ Ingress controller is ready."
}


COMMAND=${1-}
if [ -z "$COMMAND" ]; then
    echo "Usage: $0 <up|down|load>"
    exit 1
fi

case "$COMMAND" in
    up)
        echo "🚀 Creating Kind cluster '$CLUSTER_NAME'..."
        kind create cluster --name "$CLUSTER_NAME" --config "$KIND_CONFIG"
        echo "✅ Cluster created successfully."

        install_ingress_controller
        install_operator

        ;;

    down)
        echo "🔥 Deleting Kind cluster '$CLUSTER_NAME'..."
        kind delete cluster --name "$CLUSTER_NAME"
        echo "✅ Cluster deleted."
        ;;

    load)
        # Проверяем второй аргумент
        IMAGE_NAME=${2-}
        if [ -z "$IMAGE_NAME" ]; then
            echo "Error: Docker image name is required for 'load' command."
            echo "Usage: $0 load <image-name:tag>"
            exit 1
        fi
        echo "📦 Loading image '$IMAGE_NAME' into cluster..."
        kind load docker-image "$IMAGE_NAME" --name "$CLUSTER_NAME"
        echo "✅ Image loaded."
        ;;

    *)
        echo "Error: Unknown command '$COMMAND'."
        echo "Usage: $0 <up|down|load>"
        exit 1
        ;;
esac
