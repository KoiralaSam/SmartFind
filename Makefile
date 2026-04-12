PROTO_DIR := proto
PROTO_SRC := $(wildcard $(PROTO_DIR)/*.proto)
PROTO_OUT_DIR := shared/proto
PROTO_PY_OUT_DIR := shared/proto_py
PYTHON ?= python3
GO ?= go
K8S_SECRET_FILE := infra/development/k8s/secrets.yaml
DB_URL := $(shell awk -F': ' '/^[[:space:]]*DATABASE_URL:/ {print $$2}' $(K8S_SECRET_FILE) | tr -d '"')

.PHONY: proto-tools generate-proto generate-proto-python generate-proto-all migrate-create migrate-up migrate-down


generate-proto: proto-tools
	@set -eu; \
	GOPATH_BIN="$$( "$(GO)" env GOPATH )/bin"; \
	export PATH="$$GOPATH_BIN:$$PATH"; \
	for proto in $(PROTO_SRC); do \
		name=$$(basename "$$proto" .proto); \
		out_dir="$(PROTO_OUT_DIR)/$$name"; \
		mkdir -p "$$out_dir"; \
		protoc \
			--proto_path=$(PROTO_DIR) \
			--go_out="$$out_dir" \
			--go_opt=paths=source_relative \
			--go-grpc_out="$$out_dir" \
			--go-grpc_opt=paths=source_relative \
			"$$proto"; \
	done

generate-proto-python:
	@set -eu; \
	if [ -z "$(PROTO_SRC)" ]; then \
		echo "no .proto files found under $(PROTO_DIR)/"; \
		exit 1; \
	fi; \
	"$(PYTHON)" -c 'import grpc_tools' >/dev/null 2>&1 || { \
		echo "missing dependency: grpcio-tools"; \
		echo "install with: pip install grpcio-tools"; \
		exit 1; \
	}; \
	out_dir="$(PROTO_PY_OUT_DIR)/$$name"; \
	mkdir -p "$$out_dir"; \
	touch out_dir; \
	"$(PYTHON)" -m grpc_tools.protoc \
		-I"$(PROTO_DIR)" \
		--python_out="$(PROTO_PY_OUT_DIR)" \
		--pyi_out="$(PROTO_PY_OUT_DIR)" \
		--grpc_python_out="$(PROTO_PY_OUT_DIR)" \
		$(PROTO_SRC)

generate-proto-all: generate-proto generate-proto-python

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
