APP_NAME=api

.PHONY: run build test up down up-dev down-dev migrate seed

run:
	docker compose run --rm dev go run ./cmd/api

build:
	docker compose run --rm dev go build -o bin/$(APP_NAME) ./cmd/api

test:
	docker compose up -d db-test
	docker compose run --rm \
		-e TEST_DB_DSN="postgres://commerce:commerce@db-test:5432/commerce_test?sslmode=disable" \
		dev go test ./...

migrate:
	./devenv go run ./cmd/migrate

seed:
	./devenv go run ./cmd/seed

up:
	docker compose --profile prod up --build

down:
	docker compose down --remove-orphans

up-dev:
	docker compose --profile dev up --build -d

down-dev:
	docker compose --profile dev down
