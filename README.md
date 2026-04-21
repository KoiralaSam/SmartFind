# SmartFind: AI-Powered Lost & Found Management

SmartFind is a multi-agent system for lost-item recovery in public transit.  
It combines:

- A conversational intake agent (passenger reports lost items in natural language)
- A matching agent (compares lost reports with found items)
- A predictive analytics agent (identifies routes/stations with frequent losses)

## Project Overview

Manual lost-and-found workflows are slow and error-prone. SmartFind modernizes the process with a Go backend, Python AI agents, and a web frontend.

Core goals:

- Collect high-quality lost item reports through guided conversational input
- Improve match accuracy between lost reports and found inventory
- Help transit authorities prioritize hotspots using historical trend analysis

## System Components

- `web`: Passenger and operator interface (currently scaffolded as a blank starter page)
- `services`: Go backend services
- `shared`: Shared contracts and utilities for Go services
- `docs/flows`: Project-specific process and agent flow documentation
- `infra`: Kubernetes + Docker setup for development/production

## Required Tools

- Docker
- Go
- Tilt
- Kubernetes (Minikube or Docker Desktop Kubernetes)
- Node.js 20+ (for web app)

## Local Environment Setup

Follow these steps in order:

```bash
brew install minikube
kubectl config use-context docker-desktop
curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash
```

## Run Locally

Start local development:

```bash
tilt up
```

Check resources:

```bash
kubectl get pods
```

Deployment workflow details, CI/CD expectations, and rollback commands live in `docs/deployment.md`.

## Makefile Commands

The `Makefile` includes migration helpers that read `DATABASE_URL` from [infra/development/k8s/secrets.yaml](infra/development/k8s/secrets.yaml).

Install the migration CLI if it is not already available:

```bash
brew install golang-migrate
```

Create a new sequential migration:

```bash
make migrate-create name=add_users_table
```

Run all pending migrations:

```bash
make migrate-up
```

Roll back the most recent migration:

```bash
make migrate-down
```

## Web App (Blank Starter)

The frontend is intentionally reset to a blank page for the new project phase.

Run the web app directly:

```bash
cd web
npm install
npm run dev
```

## Create New Go Services

Use the service generator:

```bash
go run tools/create_service.go -name <service-name>
```

Example:

```bash
go run tools/create_service.go -name intake
```

This creates:

- `services/<service-name>-service/cmd`
- `services/<service-name>-service/internal/domain`
- `services/<service-name>-service/internal/service`
- `services/<service-name>-service/internal/infrastructure/{events,grpc,repository}`
- `services/<service-name>-service/pkg/types`
- `services/<service-name>-service/README.md`

## Flows

Current project flows are documented under `docs/flows`:

- Conversational intake flow
- Lost/found matching flow
- Predictive analytics flow

## Development Notes

- `web/src/contracts.ts`, `web/src/constants.ts`, and `web/src/types.ts` are now mock examples to guide the real implementation.
- Infra Docker definitions are reduced to web-only templates for this project reset.

## Git Workflow for Developers

Follow this workflow for every task:

1. Start from updated `main` locally and create your feature branch:

```bash
git checkout main
git pull origin main
git checkout -b feature/<your-feature-name>
```

2. Make changes on your local feature branch, commit, and push to your remote feature branch:

```bash
git add .
git commit -m "your message"
git push -u origin feature/<your-feature-name>
```

3. Open a Pull Request from your remote feature branch to remote `main`:

- Source: `feature/<your-feature-name>`
- Target: `main`

4. Keep your local branches synced after other PRs are merged:

```bash
# Update local main from remote main
git checkout main
git pull origin main

# Bring latest main into your feature branch
git checkout feature/<your-feature-name>
git merge main
```

5. If merge/update creates new changes, push again to your own remote feature branch:

```bash
git push origin feature/<your-feature-name>
```

Repeat this sync cycle so each developer continuously pulls latest `main` changes, merges into their local feature branch, and pushes updates to their own remote feature branch.
