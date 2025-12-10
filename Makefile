APP_NAME=api

.PHONY: run build test up down up-dev down-dev

run:
	go run ./cmd/api

build:
	go build -o bin/$(APP_NAME) ./cmd/api

test:
	go test ./...

up:
	docker compose --profile prod up --build

down:
	docker compose down

up-dev:
	docker compose --profile dev up --build

down-dev:
	docker compose --profile dev down
