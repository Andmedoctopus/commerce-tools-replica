APP_NAME=api

.PHONY: run build test up down

run:
	go run ./cmd/api

build:
	go build -o bin/$(APP_NAME) ./cmd/api

test:
	go test ./...

up:
	docker compose up --build

down:
	docker compose down
