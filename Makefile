PROTO_DIR := proto
PROTO_SRC := $(wildcard $(PROTO_DIR)/*.proto)
GO_OUT := .
K8S_SECRET_FILE := infra/development/k8s/secrets.yaml
DB_URL := $(shell awk -F': ' '/^[[:space:]]*DATABASE_URL:/ {print $$2}' $(K8S_SECRET_FILE) | tr -d '"')

.PHONY: generate-proto migrate-create migrate-up migrate-down
generate-proto:
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GO_OUT) \
		--go-grpc_out=$(GO_OUT) \
		$(PROTO_SRC)

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "usage: make migrate-create name=<migration_name>"; \
		exit 1; \
	fi
	migrate create -ext sql -dir migrations -seq $(name)

migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1
