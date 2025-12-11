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
- Docker Compose: base `docker-compose.yml` defines all containers (db, db-test, migrate, api, api-dev, dev) and port mappings. `docker-compose.app.yml` holds shared service definitions: `app` (builds the app image), `dev-base` (dev image/volumes/env), `db` (primary Postgres). `db-test` in the base compose extends `db` with overridden env/volume/healthcheck for an isolated test database. `migrate` extends `app` to run `golang-migrate` once before `api`/`api-dev`; it overrides entrypoint to `/srv/migrate`. `api-dev` extends `dev-base`, mounts the repo, and runs `air` for live reload. `dev` extends `dev-base` as a helper shell and depends on both db and db-test.
- Dev Go cache: bind-mounted to `./_gocache/mod` and `./_gocache/build` so `docker compose down -v` won’t erase caches.
- Seeds: `cmd/seed` applies basic demo data (project "demo" + sample products). Use `make seed` (runs inside dev container) after migrations.
- Makefile targets (planned): `make run`, `make test`, `make migrate-up/down`, `make lint`, `make seed`. `make up`/`down` target the `prod` profile; `make up-dev`/`down-dev` target the `dev` profile (starts db + api-dev + dev).
- Local env: `.env.example` for app and DB credentials; default ports for Postgres.
- Migrations: embedded SQL in `internal/migrate/sql` using `golang-migrate` (iofs + postgres driver); applied automatically on API start and via `cmd/migrate` CLI.
- Observability: structured logs; basic request metrics later.
- Health endpoints: `/healthz` and `/readyz` (Kubernetes-style suffix) reserved for probes, separate from business routes.
- Dev container: `dev` (Dockerfile `dev` target, repo mounted, bash) for running commands inside the compose network; start with `make up-dev`, and use `./devenv` to exec into it. `api-dev` runs with hot-reload (`air`) on the same profile, after `migrate` completes.
- Error handling: required context values (e.g., project/auth) are treated as invariants; handlers may panic when missing so recovery returns 500 and logs the issue instead of leaking internal details to clients.

### Near-Term Tasks
1) Scaffold Go module, folder layout (`cmd/api`, `internal/{http,service,repo,domain}`), config loading, logger.
2) Define DB schema + migrations; add Compose file with app+Postgres; wire DSN/envs.
3) Implement auth flows (signup/login/token middleware).
4) Implement product list/detail endpoints.
5) Implement cart lifecycle (create/update lines/recalc totals/get/checkout stub) with project scoping.
6) Add API docs stub (OpenAPI) and basic tests (unit + integration via docker-compose).
