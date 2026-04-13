k8s_yaml("./infra/development/k8s/app-config.yaml")
k8s_yaml("./infra/development/k8s/secrets.yaml")
k8s_yaml("./infra/development/k8s/postgres-deployment.yaml")

docker_build(
  "smartfind/web",
  ".",
  dockerfile="./infra/development/docker/web.Dockerfile",
)

docker_build(
  "smartfind/api-gateway",
  ".",
  dockerfile="./infra/development/docker/api-gateway.Dockerfile",
)

docker_build(
  "smartfind/chat-agent",
  ".",
  dockerfile="./infra/development/docker/chat-agent.Dockerfile",
)

docker_build(
  "smartfind/detail-extracter-agent",
  ".",
  dockerfile="./infra/development/docker/detail-extracter-agent.Dockerfile",
)

docker_build(
  "smartfind/predictive-analytics-agent",
  ".",
  dockerfile="./infra/development/docker/predictive-analytics-agent.Dockerfile",
)

docker_build(
  "smartfind/passenger-service",
  ".",
  dockerfile="./infra/development/docker/passenger-service.Dockerfile",
)

k8s_yaml("./infra/development/k8s/web-deployment.yaml")
k8s_yaml("./infra/development/k8s/api-gateway-deployment.yaml")
k8s_yaml("./infra/development/k8s/chat-agent-deployment.yaml")
k8s_yaml("./infra/development/k8s/detail-extracter-agent-deployment.yaml")
k8s_yaml("./infra/development/k8s/predictive-analytics-agent-deployment.yaml")
k8s_yaml("./infra/development/k8s/passenger-service-deployment.yaml")
k8s_resource("postgres", port_forwards=5432, labels="infrastructure")

local_resource(
  "db-migrate",
  cmd="""
set -eu

echo "Waiting for postgres to be ready..."
until kubectl exec deploy/postgres -- pg_isready -U smartfind -d smartfind > /dev/null 2>&1; do
  echo "Postgres not ready yet, retrying in 2s..."
  sleep 2
done

echo "Postgres is ready. Running migrations..."
migrate -path ./migrations \\
  -database "postgres://smartfind:smartfind@localhost:5432/smartfind?sslmode=disable" \\
  up

echo "Migrations completed."
""",
  deps=["./Makefile", "./migrations", "./infra/development/k8s/secrets.yaml"],
  resource_deps=["postgres"],
  labels="infrastructure",
)

k8s_resource("web", port_forwards=5173, labels="frontend", resource_deps=["db-migrate"])
k8s_resource("api-gateway", port_forwards=8081, labels="services", resource_deps=["passenger-service"])
k8s_resource("chat-agent", port_forwards=8090, labels="services", resource_deps=["db-migrate"])
k8s_resource("detail-extracter-agent", port_forwards=8091, labels="services", resource_deps=["db-migrate"])
k8s_resource("predictive-analytics-agent", port_forwards=8092, labels="services", resource_deps=["db-migrate"])
k8s_resource("passenger-service", port_forwards=50051, labels="services", resource_deps=["db-migrate"])
