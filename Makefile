.PHONY: run build test lint migrate-up migrate-down admin-create docker-build

ENV_FILE ?= .env
ADMIN_EMAIL ?= admin@example.com
ADMIN_NAME ?= Tracklume Admin

run:
	set -a; . ./$(ENV_FILE); set +a; go run ./cmd/api

build:
	go build ./cmd/api

test:
	go test ./...

lint:
	go vet ./...

migrate-up:
	set -a; . ./$(ENV_FILE); set +a; go run ./cmd/migrate up

migrate-down:
	set -a; . ./$(ENV_FILE); set +a; go run ./cmd/migrate down

admin-create:
	set -a; . ./$(ENV_FILE); set +a; go run ./cmd/admin create --email "$(ADMIN_EMAIL)" --name "$(ADMIN_NAME)"

docker-build:
	set -a; . ./build.env; set +a; docker build --file "$$DOCKERFILE" --build-arg "BUILD_IMAGE=$$BUILD_IMAGE" --build-arg "RUNTIME_IMAGE=$$RUNTIME_IMAGE" --build-arg "INSTALL_COMMAND=$$INSTALL_COMMAND" --build-arg "BUILD_COMMAND=$$BUILD_COMMAND" --build-arg "RUNTIME_PREPARE_COMMAND=$$RUNTIME_PREPARE_COMMAND" --build-arg "START_COMMAND=$$START_COMMAND" --build-arg "HEALTHCHECK_COMMAND=$$HEALTHCHECK_COMMAND" -t tracklume-api:local .
