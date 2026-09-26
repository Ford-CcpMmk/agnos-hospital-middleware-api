ifneq (,$(wildcard ./.env))
include .env
export
endif

DATABASE_URL ?= postgres://agnos:agnos@localhost:5432/agnos?sslmode=disable
GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.28.0

.PHONY: run test fmt vet build stack-up stack-down stack-logs db-up db-down db-logs migrate-up migrate-down migrate-status

run:
	go run ./cmd/api

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

build:
	go build -o bin/api ./cmd/api

stack-up:
	docker compose up -d --build

stack-down:
	docker compose down

stack-logs:
	docker compose logs -f nginx api migrate postgres hospital-a-mock

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

db-logs:
	docker compose logs -f postgres

migrate-up:
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" down

migrate-status:
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" status
