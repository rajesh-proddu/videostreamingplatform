#!/usr/bin/env bash

set -euo pipefail

NAMESPACE="${NAMESPACE:-videostreamingplatform}"
OBS_NAMESPACE="${OBS_NAMESPACE:-observability}"
DEFAULT_SELECTOR="${DEFAULT_SELECTOR:-app=metadata-service}"
DEFAULT_TIMEOUT="${DEFAULT_TIMEOUT:-900}"

usage() {
  cat <<'EOF'
Usage:
  scripts/aws/resiliency-node-cycle.sh snapshot
  scripts/aws/resiliency-node-cycle.sh drain --node <node-name>
  scripts/aws/resiliency-node-cycle.sh uncordon --node <node-name>
  scripts/aws/resiliency-node-cycle.sh kind-stop --node <node-name>
  scripts/aws/resiliency-node-cycle.sh kind-start --node <node-name>
  scripts/aws/resiliency-node-cycle.sh terminate --node <node-name> [--region <aws-region>]
  scripts/aws/resiliency-node-cycle.sh wait-pods [--selector <label-selector>] [--namespace <ns>] [--expected-ready <count>] [--timeout <seconds>]

Examples:
  scripts/aws/resiliency-node-cycle.sh snapshot
  scripts/aws/resiliency-node-cycle.sh drain --node ip-10-0-12-220.ec2.internal
  scripts/aws/resiliency-node-cycle.sh kind-stop --node videostreamingplatform-worker
  scripts/aws/resiliency-node-cycle.sh kind-start --node videostreamingplatform-worker
  scripts/aws/resiliency-node-cycle.sh terminate --node ip-10-0-12-220.ec2.internal --region us-east-1
  scripts/aws/resiliency-node-cycle.sh wait-pods --selector app=metadata-service --expected-ready 2 --timeout 900
EOF
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

provider_instance_id() {
  local node_name="$1"
  local provider_id
  provider_id="$(kubectl get node "$node_name" -o jsonpath='{.spec.providerID}')"
  echo "${provider_id##*/}"
}

snapshot() {
  echo "=== Nodes ==="
  kubectl get nodes -o wide
  echo
  echo "=== Application Pods (${NAMESPACE}) ==="
  kubectl get pods -n "$NAMESPACE" -o wide
  echo
  echo "=== Observability Pods (${OBS_NAMESPACE}) ==="
  kubectl get pods -n "$OBS_NAMESPACE" -o wide 2>/dev/null || true
  echo
  echo "=== Recent Events ==="
  kubectl get events -A --sort-by=.lastTimestamp | tail -50 || true
}

drain_node() {
  local node_name="$1"
  echo "Draining node: ${node_name}"
  kubectl drain "$node_name" --ignore-daemonsets --delete-emptydir-data --force --grace-period=30 --timeout=10m
}

uncordon_node() {
  local node_name="$1"
  echo "Uncordoning node: ${node_name}"
  kubectl uncordon "$node_name"
}

kind_stop_node() {
  local node_name="$1"
  require_cmd docker
  echo "Stopping Kind node container: ${node_name}"
  docker stop "$node_name"
}

kind_start_node() {
  local node_name="$1"
  require_cmd docker
  echo "Starting Kind node container: ${node_name}"
  docker start "$node_name"
}

terminate_node() {
  local node_name="$1"
  local region="$2"
  local instance_id
  instance_id="$(provider_instance_id "$node_name")"
  echo "Terminating node ${node_name} (instance ${instance_id}) in region ${region}"
  aws ec2 terminate-instances --instance-ids "$instance_id" --region "$region"
}

wait_pods() {
  local selector="$1"
  local namespace="$2"
  local expected_ready="$3"
  local timeout_seconds="$4"
  local start_ts current ready total

  start_ts="$(date +%s)"

  echo "Waiting for ${expected_ready} Ready pods in namespace=${namespace} selector=${selector}"

  while true; do
    current="$(kubectl get pods -n "$namespace" -l "$selector" --no-headers 2>/dev/null || true)"
    total="$(printf '%s\n' "$current" | awk 'NF>0 {count++} END {print count+0}')"
    ready="$(printf '%s\n' "$current" | awk 'NF>0 {split($2,a,"/"); if (a[1] == a[2]) count++} END {print count+0}')"

    echo "ready=${ready} total=${total}"
    if [[ "$ready" -ge "$expected_ready" ]]; then
      return 0
    fi

    if (( $(date +%s) - start_ts >= timeout_seconds )); then
      echo "timed out waiting for pods to recover" >&2
      kubectl get pods -n "$namespace" -l "$selector" -o wide || true
      return 1
    fi

    sleep 10
  done
}

main() {
  require_cmd kubectl

  if [[ $# -lt 1 ]]; then
    usage
    exit 1
  fi

  local command="$1"
  shift

  case "$command" in
    snapshot)
      snapshot
      ;;
    drain)
      local node_name=""
      while [[ $# -gt 0 ]]; do
        case "$1" in
          --node) node_name="$2"; shift 2 ;;
          *) echo "unknown argument: $1" >&2; exit 1 ;;
        esac
      done
      [[ -n "$node_name" ]] || { echo "--node is required" >&2; exit 1; }
      drain_node "$node_name"
      ;;
    uncordon)
      local node_name=""
      while [[ $# -gt 0 ]]; do
        case "$1" in
          --node) node_name="$2"; shift 2 ;;
          *) echo "unknown argument: $1" >&2; exit 1 ;;
        esac
      done
      [[ -n "$node_name" ]] || { echo "--node is required" >&2; exit 1; }
      uncordon_node "$node_name"
      ;;
    kind-stop)
      local node_name=""
      while [[ $# -gt 0 ]]; do
        case "$1" in
          --node) node_name="$2"; shift 2 ;;
          *) echo "unknown argument: $1" >&2; exit 1 ;;
        esac
      done
      [[ -n "$node_name" ]] || { echo "--node is required" >&2; exit 1; }
      kind_stop_node "$node_name"
      ;;
    kind-start)
      local node_name=""
      while [[ $# -gt 0 ]]; do
        case "$1" in
          --node) node_name="$2"; shift 2 ;;
          *) echo "unknown argument: $1" >&2; exit 1 ;;
        esac
      done
      [[ -n "$node_name" ]] || { echo "--node is required" >&2; exit 1; }
      kind_start_node "$node_name"
      ;;
    terminate)
      require_cmd aws
      local node_name=""
      local region="${AWS_REGION:-us-east-1}"
      while [[ $# -gt 0 ]]; do
        case "$1" in
          --node) node_name="$2"; shift 2 ;;
          --region) region="$2"; shift 2 ;;
          *) echo "unknown argument: $1" >&2; exit 1 ;;
        esac
      done
      [[ -n "$node_name" ]] || { echo "--node is required" >&2; exit 1; }
      terminate_node "$node_name" "$region"
      ;;
    wait-pods)
      local selector="$DEFAULT_SELECTOR"
      local namespace="$NAMESPACE"
      local expected_ready="2"
      local timeout_seconds="$DEFAULT_TIMEOUT"
      while [[ $# -gt 0 ]]; do
        case "$1" in
          --selector) selector="$2"; shift 2 ;;
          --namespace) namespace="$2"; shift 2 ;;
          --expected-ready) expected_ready="$2"; shift 2 ;;
          --timeout) timeout_seconds="$2"; shift 2 ;;
          *) echo "unknown argument: $1" >&2; exit 1 ;;
        esac
      done
      wait_pods "$selector" "$namespace" "$expected_ready" "$timeout_seconds"
      ;;
    -h|--help|help)
      usage
      ;;
    *)
      echo "unknown command: ${command}" >&2
      usage
      exit 1
      ;;
  esac
}

main "$@"
