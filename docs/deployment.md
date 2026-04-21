## SmartFind Deployment

This repository uses three layers for delivery:

- `Tilt` for local Kubernetes development.
- `GitHub Actions` for CI/CD.
- `kubectl apply` plus `kubectl rollout status` for cluster deployments.

### Canonical services

These are the services built and pushed by CI:

- `web`
  - Dockerfile: `infra/development/docker/web.Dockerfile`
  - Image: `ghcr.io/<owner>/smartfind/web:<git-sha>`
- `api-gateway`
  - Dockerfile: `infra/development/docker/api-gateway.Dockerfile`
  - Image: `ghcr.io/<owner>/smartfind/api-gateway:<git-sha>`
- `chat-agent`
  - Dockerfile: `infra/development/docker/chat-agent.Dockerfile`
  - Image: `ghcr.io/<owner>/smartfind/chat-agent:<git-sha>`
- `detail-extracter-agent`
  - Dockerfile: `infra/development/docker/detail-extracter-agent.Dockerfile`
  - Image: `ghcr.io/<owner>/smartfind/detail-extracter-agent:<git-sha>`
- `predictive-analytics-agent`
  - Dockerfile: `infra/development/docker/predictive-analytics-agent.Dockerfile`
  - Image: `ghcr.io/<owner>/smartfind/predictive-analytics-agent:<git-sha>`
- `passenger-service`
  - Dockerfile: `infra/development/docker/passenger-service.Dockerfile`
  - Image: `ghcr.io/<owner>/smartfind/passenger-service:<git-sha>`
- `passenger-match-worker`
  - Dockerfile: `infra/development/docker/passenger-match-worker.Dockerfile`
  - Image: `ghcr.io/<owner>/smartfind/passenger-match-worker:<git-sha>`
- `staff-service`
  - Dockerfile: `infra/development/docker/staff-service.Dockerfile`
  - Image: `ghcr.io/<owner>/smartfind/staff-service:<git-sha>`

The deploy scripts use `scripts/deploy/common.sh` as the single source of truth for buildable services, rollout-checked deployments, and cronjobs.

### Local development with Tilt

Prerequisites:

- Docker
- `kubectl`
- a local Kubernetes context such as Docker Desktop Kubernetes or Minikube
- `tilt`
- Go 1.25+
- Node.js 20+

Start the local stack:

```bash
tilt up
```

Stop the local stack:

```bash
tilt down
```

Tilt uses the development manifests in `infra/development/k8s` and local image names like `smartfind/api-gateway`.

### GitHub Actions pipeline

The workflow in `.github/workflows/ci-cd.yml` does the following:

1. Runs validation on pull requests and pushes:
   - `go test ./...`
   - `npm ci`
   - `npm run lint`
   - `npm run build`
2. Builds and pushes all service images to `ghcr.io`.
3. Deploys `main` to the `development` GitHub environment.
4. Deploys version tags such as `v1.0.0` to the `production` GitHub environment.

`latest` tags are only pushed from `main`. Immutable deployments should use the commit SHA tag.

### Required GitHub configuration

Create two GitHub environments:

- `development`
- `production`

For each environment, configure:

- Secret: `KUBE_CONFIG_DATA`
  - base64-encoded kubeconfig for the target cluster/context
- Variable: `K8S_NAMESPACE`
  - optional override; defaults to `smartfind-development` or `smartfind-production`

For production, protect the GitHub environment with required reviewers to create a manual approval gate.

### Cluster prerequisites

Before CI deploys the workloads, the target namespace should already contain any runtime secrets the services expect, such as:

- `postgres-secret`
- `internal-service-secret`
- `passenger-service-secrets`
- `staff-service-secrets`
- `openai-api-key`
- `deepgram-secrets`
- `analytics-agent`
- `detail-extracter-agent`
- `s3-credentials`
- `mailtrap-secrets`

If GHCR packages are private, also create an image pull secret and attach it to the namespace service account.

### Manual deployment

Deploy development manifests with a specific image tag:

```bash
scripts/deploy/apply.sh \
  --environment development \
  --namespace smartfind-development \
  --registry ghcr.io/koiralasam/smartfind \
  --image-tag <git-sha>
```

Deploy production manifests with a specific image tag:

```bash
scripts/deploy/apply.sh \
  --environment production \
  --namespace smartfind-production \
  --registry ghcr.io/koiralasam/smartfind \
  --image-tag <git-sha>
```

The deploy script:

- creates the namespace if needed
- applies the selected manifest set with `kubectl apply -f`
- updates images with `kubectl set image`
- waits for each deployment with `kubectl rollout status`
- verifies the cronjob exists
- prints a summary of deployments, services, and pods

### Rollback

Roll back a deployment:

```bash
scripts/deploy/rollback.sh \
  --namespace smartfind-production \
  --deployment api-gateway
```

### Notes

- The development Tilt flow remains the fastest inner loop for local work.
- Production manifests mirror the current application layout and are intentionally lightweight for school projects, MVPs, and small teams.
- The old `trip-service` manifests are not part of the CI/deploy inventory and are intentionally excluded by the deploy scripts.
