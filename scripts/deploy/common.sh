#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

readonly BUILD_SERVICES=(
  "web"
  "api-gateway"
  "chat-agent"
  "detail-extracter-agent"
  "predictive-analytics-agent"
  "passenger-service"
  "passenger-match-worker"
  "staff-service"
)

readonly ROLLOUT_DEPLOYMENTS=(
  "web"
  "api-gateway"
  "chat-agent"
  "detail-extracter-agent"
  "predictive-analytics-agent"
  "passenger-service"
  "staff-service"
)

readonly CRONJOBS=(
  "passenger-match-worker"
)

service_dockerfile() {
  case "$1" in
    web) echo "infra/development/docker/web.Dockerfile" ;;
    api-gateway) echo "infra/development/docker/api-gateway.Dockerfile" ;;
    chat-agent) echo "infra/development/docker/chat-agent.Dockerfile" ;;
    detail-extracter-agent) echo "infra/development/docker/detail-extracter-agent.Dockerfile" ;;
    predictive-analytics-agent) echo "infra/development/docker/predictive-analytics-agent.Dockerfile" ;;
    passenger-service) echo "infra/development/docker/passenger-service.Dockerfile" ;;
    passenger-match-worker) echo "infra/development/docker/passenger-match-worker.Dockerfile" ;;
    staff-service) echo "infra/development/docker/staff-service.Dockerfile" ;;
    *)
      echo "unknown service: $1" >&2
      return 1
      ;;
  esac
}

manifest_dir() {
  local environment="${1:?environment is required}"
  echo "${ROOT_DIR}/infra/${environment}/k8s"
}

manifest_files() {
  local environment="${1:?environment is required}"

  case "$environment" in
    development)
      cat <<EOF
$(manifest_dir development)/app-config.yaml
$(manifest_dir development)/web-deployment.yaml
$(manifest_dir development)/api-gateway-deployment.yaml
$(manifest_dir development)/chat-agent-deployment.yaml
$(manifest_dir development)/detail-extracter-agent-deployment.yaml
$(manifest_dir development)/predictive-analytics-agent-deployment.yaml
$(manifest_dir development)/passenger-service-deployment.yaml
$(manifest_dir development)/staff-service-deployment.yaml
$(manifest_dir development)/passenger-match-worker-cronjob.yaml
EOF
      ;;
    production)
      cat <<EOF
$(manifest_dir production)/app-config.yaml
$(manifest_dir production)/web-deployment.yaml
$(manifest_dir production)/api-gateway-deployment.yaml
$(manifest_dir production)/chat-agent-deployment.yaml
$(manifest_dir production)/detail-extracter-agent-deployment.yaml
$(manifest_dir production)/predictive-analytics-agent-deployment.yaml
$(manifest_dir production)/passenger-service-deployment.yaml
$(manifest_dir production)/staff-service-deployment.yaml
$(manifest_dir production)/passenger-match-worker-cronjob.yaml
EOF
      ;;
    *)
      echo "unsupported environment: ${environment}" >&2
      return 1
      ;;
  esac
}

image_ref() {
  local registry="${1%/}"
  local service="${2:?service is required}"
  local tag="${3:?tag is required}"

  echo "${registry}/${service}:${tag}"
}
