#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/deploy/rollback.sh --deployment <name> [--namespace <namespace>]

Example:
  scripts/deploy/rollback.sh --namespace smartfind-production --deployment api-gateway
EOF
}

deployment=""
namespace="default"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --deployment)
      deployment="${2:-}"
      shift 2
      ;;
    --namespace)
      namespace="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ -z "${deployment}" ]]; then
  usage
  exit 1
fi

kubectl rollout undo -n "${namespace}" "deployment/${deployment}"
kubectl rollout status -n "${namespace}" "deployment/${deployment}" --timeout=180s
