.PHONY: run build test lint migrate-up migrate-down seed docker-build

ENV_FILE ?= .env

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

docker-build:
	docker build -t tracklume-api:local .
