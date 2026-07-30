.PHONY: run build test lint migrate-up migrate-down seed admin-create docker-build

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

seed:
	set -a; . ./$(ENV_FILE); set +a; go run ./cmd/seed

admin-create:
	set -a; . ./$(ENV_FILE); set +a; go run ./cmd/admin create --email "$(ADMIN_EMAIL)" --name "$(ADMIN_NAME)"

docker-build:
	docker build -t tracklume-api:local .
