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
- Docker Compose: `docker-compose.app.yml` holds shared service definitions (`app` build, `dev-base` dev image/volumes/env, `db`). Top-level `docker-compose.yml` wires everything with ports/depends_on and adds `db-test` (extends `db` with its own env/volume/healthcheck), `migrate` (extends `app`, entrypoint `/srv/migrate`, runs before API), `api` (prod server on host `8080`), `api-dev` (hot reload via `air`, exposed on host `8081` to avoid port clashes), `dev` (helper shell), and `pgadmin` on host `5050` preloaded with connections to `db` and `db-test` (passwords saved; master password prompt disabled). No profiles; `make up`/`down` manage the full stack.
- Dev Go cache: bind-mounted to `./_gocache/mod` and `./_gocache/build` so `docker compose down -v` won’t erase caches.
- Seeds: `cmd/seed` applies basic demo data (project "demo" + sample products). Use `make seed` (runs inside dev container) after migrations.
- Makefile: `make run` (dev go run), `make build`, `make test` (brings up `db-test`, runs `go test` inside `dev`), `make migrate`, `make seed`, `make up`/`down` to start/stop services.
- Local env: `.env.example` for app and DB credentials; default ports for Postgres.
- Migrations: embedded SQL in `internal/migrate/sql` using `golang-migrate` (iofs + postgres driver); applied via `cmd/migrate` and the `migrate` compose service (before `api`/`api-dev`).
- Observability: structured logs; basic request metrics later.
- Health endpoints: `/healthz` and `/readyz` (Kubernetes-style suffix) reserved for probes, separate from business routes.
- Dev container: `dev` (Dockerfile `dev` target, repo mounted, bash) for running commands inside the compose network; start `dev`/`api-dev` via compose and use `./devenv` to exec. Required context values (e.g., project/auth) are treated as invariants; handlers may panic when missing so recovery returns 500 and logs the issue instead of leaking internal details to clients.
- Command output etiquette: avoid noisy commands (e.g., `go mod tidy`) unless required for builds; keep terminal output concise when possible.
- Do not run `gofmt` or formatters on SQL migration files; edit them directly to avoid unintended changes.

### Near-Term Tasks
1) Scaffold Go module, folder layout (`cmd/api`, `internal/{http,service,repo,domain}`), config loading, logger.
2) Define DB schema + migrations; add Compose file with app+Postgres; wire DSN/envs.
3) Implement auth flows (signup/login/token middleware).
4) Implement product list/detail endpoints.
5) Implement cart lifecycle (create/update lines/recalc totals/get/checkout stub) with project scoping.
6) Add API docs stub (OpenAPI) and basic tests (unit + integration via docker-compose).

### CSV Importer Usage
- Import commercetools product export CSV: `./devenv go run ./cmd/importer -file imports/Products_Export_09-12-25_19-36.csv -project <project-key>`.
- If the project key is missing, the importer will create it with name equal to the key.
- The importer resolves/creates the project and upserts products (key/SKU/name/description/price/currency plus image URLs stored in `attributes.images`).
