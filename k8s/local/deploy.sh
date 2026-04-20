#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
K8S_LOCAL="$REPO_ROOT/k8s/local"
TERRAFORM_DIR="$K8S_LOCAL/terraform"
CLUSTER_NAME="videostreamingplatform"
NAMESPACE="videostreamingplatform"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
err()  { echo -e "${RED}[ERROR]${NC} $*"; }

usage() {
  echo "Usage: $0 [up|down|status|rebuild]"
  echo "  up      - Create cluster, build images, deploy all services"
  echo "  down    - Tear down the cluster"
  echo "  status  - Show cluster and pod status"
  echo "  rebuild - Rebuild and redeploy service images only"
  exit 1
}

check_prerequisites() {
  local missing=()
  for cmd in docker kubectl terraform kind; do
    if ! command -v "$cmd" &>/dev/null; then
      missing+=("$cmd")
    fi
  done
  if [ ${#missing[@]} -gt 0 ]; then
    err "Missing required tools: ${missing[*]}"
    exit 1
  fi
}

create_cluster() {
  if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    log "KIND cluster '$CLUSTER_NAME' already exists"
    kubectl cluster-info --context "kind-${CLUSTER_NAME}" &>/dev/null || true
    return 0
  fi

  log "Creating KIND cluster with Terraform..."
  cd "$TERRAFORM_DIR"
  terraform init -input=false
  terraform apply -auto-approve -input=false
  cd "$REPO_ROOT"

  log "Setting kubectl context..."
  kubectl cluster-info --context "kind-${CLUSTER_NAME}"
}

build_and_load_images() {
  log "Building service images..."
  docker build -t videostreamingplatform/metadata-service:latest \
    -f "$REPO_ROOT/build/docker/metadataservice.Dockerfile" "$REPO_ROOT"
  docker build -t videostreamingplatform/data-service:latest \
    -f "$REPO_ROOT/build/docker/dataservice.Dockerfile" "$REPO_ROOT"
  docker build -t videostreamingplatform/cdn-invalidator:latest \
    -f "$REPO_ROOT/build/docker/cdn-invalidator.Dockerfile" "$REPO_ROOT"

  log "Loading images into KIND cluster..."
  kind load docker-image videostreamingplatform/metadata-service:latest --name "$CLUSTER_NAME"
  kind load docker-image videostreamingplatform/data-service:latest --name "$CLUSTER_NAME"
  kind load docker-image videostreamingplatform/cdn-invalidator:latest --name "$CLUSTER_NAME"
}

create_grafana_dashboards_configmap() {
  local dashboards_dir="$REPO_ROOT/scripts/grafana/dashboards"
  if [ ! -d "$dashboards_dir" ]; then
    warn "No Grafana dashboards directory found, skipping configmap"
    return 0
  fi

  log "Creating Grafana dashboards configmap..."
  kubectl create configmap grafana-dashboards \
    --namespace=observability \
    --from-file="$dashboards_dir" \
    --dry-run=client -o yaml | kubectl apply -f -
}

deploy_manifests() {
  log "Deploying Kubernetes manifests..."

  # Namespace first
  kubectl apply -f "$K8S_LOCAL/namespace.yaml"

  # Secrets and config
  kubectl apply -f "$K8S_LOCAL/secrets.yaml"
  kubectl apply -f "$K8S_LOCAL/mysql-init.yaml"
  kubectl apply -f "$K8S_LOCAL/configmap.yaml"

  # Infrastructure
  kubectl apply -f "$K8S_LOCAL/mysql.yaml"
  kubectl apply -f "$K8S_LOCAL/minio.yaml"
  kubectl apply -f "$K8S_LOCAL/redis.yaml"
  kubectl apply -f "$K8S_LOCAL/kafka.yaml"
  kubectl apply -f "$K8S_LOCAL/cdn-proxy.yaml"

  # LocalStack (Glue + S3 for Iceberg)
  kubectl apply -f "$K8S_LOCAL/localstack.yaml"

  # Observability
  kubectl apply -f "$K8S_LOCAL/observability.yaml"
  create_grafana_dashboards_configmap

  # Wait for infra to be ready
  log "Waiting for MySQL to be ready..."
  kubectl wait --namespace="$NAMESPACE" \
    --for=condition=ready pod -l app=mysql \
    --timeout=120s || warn "MySQL not ready yet, continuing..."

  log "Waiting for MinIO to be ready..."
  kubectl wait --namespace="$NAMESPACE" \
    --for=condition=ready pod -l app=minio \
    --timeout=90s || warn "MinIO not ready yet, continuing..."

  log "Waiting for Redis to be ready..."
  kubectl wait --namespace="$NAMESPACE" \
    --for=condition=ready pod -l app=redis \
    --timeout=60s || warn "Redis not ready yet, continuing..."

  # Application services
  kubectl apply -f "$K8S_LOCAL/services.yaml"
  kubectl apply -f "$K8S_LOCAL/metadata-service-deploy.yaml"
  kubectl apply -f "$K8S_LOCAL/data-service-deploy.yaml"
  kubectl apply -f "$K8S_LOCAL/cdn-invalidator-deploy.yaml"

  log "Waiting for services to be ready..."
  kubectl wait --namespace="$NAMESPACE" \
    --for=condition=ready pod -l app=metadata-service \
    --timeout=120s || warn "metadata-service not ready yet"

  kubectl wait --namespace="$NAMESPACE" \
    --for=condition=ready pod -l app=data-service \
    --timeout=120s || warn "data-service not ready yet"

  # Iceberg (REST catalog + MinIO bucket init)
  log "Waiting for Iceberg REST catalog to be ready..."
  kubectl wait --namespace=infra \
    --for=condition=ready pod -l app=iceberg-rest-catalog \
    --timeout=120s || warn "Iceberg REST catalog not ready yet"

  log "Waiting for Iceberg MinIO bucket init..."
  kubectl wait --namespace=infra \
    --for=condition=complete job/iceberg-minio-init \
    --timeout=120s || warn "Iceberg MinIO init not complete yet"

  # Analytics (watch-history-consumer + catalog-admin)
  log "Deploying analytics stack..."
  kubectl apply -f "$K8S_LOCAL/analytics.yaml"

  log "Waiting for catalog-init job..."
  kubectl wait --namespace=analytics \
    --for=condition=complete job/catalog-init \
    --timeout=120s || warn "catalog-init not complete yet"
}

show_status() {
  log "Cluster: kind-${CLUSTER_NAME}"
  echo ""
  log "Namespace: $NAMESPACE"
  kubectl get pods -n "$NAMESPACE" -o wide 2>/dev/null || warn "Namespace not found"
  echo ""
  kubectl get svc -n "$NAMESPACE" 2>/dev/null || true
  echo ""
  log "Namespace: observability"
  kubectl get pods -n observability -o wide 2>/dev/null || warn "Namespace not found"
  echo ""
  kubectl get svc -n observability 2>/dev/null || true
  echo ""
  log "Access URLs (via KIND NodePort mappings):"
  echo "  Metadata Service: http://localhost:8080"
  echo "  Data Service:     http://localhost:8081"
  echo "  Prometheus:       http://localhost:9090"
  echo "  Jaeger UI:        http://localhost:16686"
  echo "  Grafana:          http://localhost:3000 (admin/admin)"
}

teardown() {
  log "Tearing down cluster..."
  cd "$TERRAFORM_DIR"
  if [ -f "terraform.tfstate" ]; then
    terraform destroy -auto-approve -input=false
  else
    kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
  fi
  cd "$REPO_ROOT"
  log "Cluster destroyed"
}

rebuild() {
  build_and_load_images
  log "Restarting deployments..."
  kubectl rollout restart deployment/metadata-service -n "$NAMESPACE"
  kubectl rollout restart deployment/data-service -n "$NAMESPACE"
  kubectl rollout status deployment/metadata-service -n "$NAMESPACE" --timeout=120s
  kubectl rollout status deployment/data-service -n "$NAMESPACE" --timeout=120s
  show_status
}

# Main
check_prerequisites

case "${1:-}" in
  up)
    create_cluster
    build_and_load_images
    deploy_manifests
    show_status
    ;;
  down)
    teardown
    ;;
  status)
    show_status
    ;;
  rebuild)
    rebuild
    ;;
  *)
    usage
    ;;
esac
