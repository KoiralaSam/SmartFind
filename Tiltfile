k8s_yaml("./infra/development/k8s/app-config.yaml")
k8s_yaml("./infra/development/k8s/secrets.yaml")
k8s_yaml("./infra/development/k8s/postgres-deployment.yaml")

docker_build(
  "smartfind/web",
  ".",
  dockerfile="./infra/development/docker/web.Dockerfile",
)

docker_build(
  "smartfind/chat-agent",
  ".",
  dockerfile="./infra/development/docker/chat-agent.Dockerfile",
)

k8s_yaml("./infra/development/k8s/web-deployment.yaml")
k8s_yaml("./infra/development/k8s/chat-agent-deployment.yaml")
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
k8s_resource("chat-agent", port_forwards=8090, labels="services", resource_deps=["db-migrate"])
