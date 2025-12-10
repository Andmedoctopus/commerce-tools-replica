## commercetools-lite API Plan

Goal: build a partial, commercetools-compatible web API (projects, products, carts, auth) in Go with Postgres. Everything runs via Docker Compose (app + Postgres) for local/dev.

### Scope & Compatibility
- Multitenant: `projects` partition data; all queries/actions scoped by project key.
- Entities: projects, customers/auth, products, carts (lines, totals). Orders/discounts/inventory deferred.
- Endpoints (v1, project-scoped `/projects/{projectKey}/...`):
  - Auth: `POST /customers` (signup), `POST /login` (token), `GET /me`.
  - Products: `GET /products`, `GET /products/{id}` (optional filter by key/sku).
  - Carts: `POST /carts` (anonymous or customer), `GET /carts/{id}`, `POST /carts/{id}` (update actions: add/remove/change qty, set customer), `POST /carts/{id}/checkout` placeholder.
- Error model: consistent JSON envelope; validate IDs and project scope.

### Architecture
- Go monolith with clear layers: HTTP transport → handlers → services/use-cases → repositories (Postgres).
- Config via env; structured logging; UTC timestamps; UUID IDs.
- Migrations: `golang-migrate` (or similar). Seed fixtures for local dev.
- Testing: unit tests for services; integration tests with Postgres in Docker.
- API contract: OpenAPI stub or Postman collection (to be generated).

### Data Model (initial)
- `projects`: id, key, name, created_at.
- `customers`: id, project_id FK, email (unique per project), password_hash, created_at.
- `products`: id, project_id FK, key/sku, name, description, price_cents, currency, attributes JSONB, created_at.
- `carts`: id, project_id FK, customer_id nullable FK, currency, total_cents, created_at, state.
- `cart_lines`: id, cart_id FK, product_id FK, quantity, unit_price_cents, total_cents, snapshot JSONB.

### Auth
- Per-project bearer tokens (JWT or signed opaque). Flows: signup/login (email+password), token issuance, middleware enforcing project scope. Anonymous carts allowed without auth via cart ID (optionally `anonymousId`).
- Password hashing via bcrypt/argon2; store hash only.

### Dev/Infra
- Docker Compose: Postgres service + app service (Go binary) with env-configured DSN. App depends_on DB and runs migrations on start.
- Makefile targets (planned): `make run`, `make test`, `make migrate-up/down`, `make lint`, `make seed`.
- Local env: `.env.example` for app and DB credentials; default ports for Postgres.
- Observability: structured logs; basic request metrics later.

### Near-Term Tasks
1) Scaffold Go module, folder layout (`cmd/api`, `internal/{http,service,repo,domain}`), config loading, logger.
2) Define DB schema + migrations; add Compose file with app+Postgres; wire DSN/envs.
3) Implement auth flows (signup/login/token middleware).
4) Implement product list/detail endpoints.
5) Implement cart lifecycle (create/update lines/recalc totals/get/checkout stub) with project scoping.
6) Add API docs stub (OpenAPI) and basic tests (unit + integration via docker-compose).
