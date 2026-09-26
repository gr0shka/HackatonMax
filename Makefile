.PHONY: all build test clean docker-up docker-down migrate-up migrate-down

DB_DSN ?= "postgres://postgres:postgres@localhost:5432/route_optimizer?sslmode=disable"
MIGRATIONS_DIR ?= ./migrations

all: test build

build:
	go build -v ./...

test:
	go test -v -race ./...

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down


migrate-up:
	go run github.com/pressly/goose/v3/cmd/goose@v3.24.1 -dir $(MIGRATIONS_DIR) postgres $(DB_DSN) up

migrate-down:
	go run github.com/pressly/goose/v3/cmd/goose@v3.24.1 -dir $(MIGRATIONS_DIR) postgres $(DB_DSN) down

migrate-status:
	go run github.com/pressly/goose/v3/cmd/goose@v3.24.1 -dir $(MIGRATIONS_DIR) postgres $(DB_DSN) status
