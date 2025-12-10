APP_NAME=api

.PHONY: run build test compose-up compose-down

run:
	go run ./cmd/api

build:
	go build -o bin/$(APP_NAME) ./cmd/api

test:
	go test ./...

compose-up:
	docker compose up --build

compose-down:
	docker compose down
