#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

usage() {
  cat <<'EOF'
Usage:
  scripts/deploy/apply.sh --environment <development|production> --registry <image-registry-prefix> --image-tag <tag> [--namespace <namespace>] [--timeout <duration>]

Example:
  scripts/deploy/apply.sh \
    --environment development \
    --namespace smartfind-development \
    --registry ghcr.io/koiralasam/smartfind \
    --image-tag abc1234
EOF
}

environment=""
namespace=""
registry=""
image_tag=""
timeout="180s"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --environment)
      environment="${2:-}"
      shift 2
      ;;
    --namespace)
      namespace="${2:-}"
      shift 2
      ;;
    --registry)
      registry="${2:-}"
      shift 2
      ;;
    --image-tag)
      image_tag="${2:-}"
      shift 2
      ;;
    --timeout)
      timeout="${2:-}"
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

if [[ -z "${environment}" || -z "${registry}" || -z "${image_tag}" ]]; then
  usage
  exit 1
fi

if [[ -z "${namespace}" ]]; then
  namespace="smartfind-${environment}"
fi

echo "Deploying SmartFind"
echo "  environment: ${environment}"
echo "  namespace:   ${namespace}"
echo "  registry:    ${registry}"
echo "  image tag:   ${image_tag}"

kubectl get namespace "${namespace}" >/dev/null 2>&1 || kubectl create namespace "${namespace}"

while IFS= read -r manifest; do
  [[ -n "${manifest}" ]] || continue
  echo "Applying ${manifest#${ROOT_DIR}/}"
  kubectl apply -n "${namespace}" -f "${manifest}"
done < <(manifest_files "${environment}")

for service in "${ROLLOUT_DEPLOYMENTS[@]}"; do
  image="$(image_ref "${registry}" "${service}" "${image_tag}")"
  echo "Updating deployment/${service} -> ${image}"
  kubectl set image -n "${namespace}" "deployment/${service}" "${service}=${image}"
done

for cronjob in "${CRONJOBS[@]}"; do
  image="$(image_ref "${registry}" "${cronjob}" "${image_tag}")"
  echo "Updating cronjob/${cronjob} -> ${image}"
  kubectl set image -n "${namespace}" "cronjob/${cronjob}" "${cronjob}=${image}"
done

for deployment in "${ROLLOUT_DEPLOYMENTS[@]}"; do
  echo "Waiting for deployment/${deployment}"
  kubectl rollout status -n "${namespace}" "deployment/${deployment}" --timeout="${timeout}"
done

if ((${#CRONJOBS[@]} > 0)); then
  echo "Checking cronjobs"
  kubectl get cronjob -n "${namespace}" "${CRONJOBS[@]}"
fi

echo "Deployment summary"
kubectl get deploy,svc,pods -n "${namespace}"
