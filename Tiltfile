k8s_yaml("./infra/development/k8s/app-config.yaml")

local_resource(
  "postgres",
  cmd="docker compose -f ./infra/development/docker/docker-compose.yml up -d postgres",
  deps=["./infra/development/docker/docker-compose.yml"],
  labels="infrastructure",
)

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
k8s_yaml("./infra/development/k8s/secrets.yaml")
k8s_yaml("./infra/development/k8s/chat-agent-deployment.yaml")
k8s_resource("web", port_forwards=5173, labels="frontend", resource_deps=["postgres"])
k8s_resource("chat-agent", port_forwards=8090, labels="services", resource_deps=["postgres"])
