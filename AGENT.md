## commercetools-replica current state

Partial commercetools-compatible web API (Go + Postgres). Project-scoped routes live under `/:projectKey`.

### Scope & Compatibility
- Multitenant: all reads/writes are scoped by project key.
- Entities: projects (DB only), customers + tokens, products, categories, carts (lines + totals).
- CT-shaped responses for customers, products, categories, carts (with some gaps noted below).

### HTTP API
- Health: `GET /healthz`, `GET /readyz`.
- Auth:
  - `POST /oauth/:projectKey/customers/token` (form-encoded, `grant_type=password`, scope `manage_project:<key>`).
  - `POST /oauth/:projectKey/anonymous/token` (form-encoded, `grant_type=client_credentials`).
  - `POST /oauth/token` returns a static stub response.
- Customers: `POST /:projectKey/me/signup`, `POST /:projectKey/me/login` (customer + active cart, no tokens), `GET /:projectKey/me` (bearer token).
- Products: `GET /:projectKey/products`, `GET /:projectKey/products/:id`, `POST /:projectKey/products/search`.
- Categories: `GET /:projectKey/categories` (limit/offset).
- Carts:
  - Raw cart shape: `POST /:projectKey/carts`, `GET /:projectKey/carts/:id`.
  - CT-style carts: `POST /:projectKey/me/carts`, `POST /:projectKey/me/carts/:id`, `DELETE /:projectKey/me/carts/:id`, `GET /:projectKey/me/active-cart`.
- Product discounts: `GET /:projectKey/product-discounts` (static demo list).

### Search behavior
- Filters: price range on `variants.prices.centAmount` and exact `categories` filter (accepts category id or key).
- Sort: `name` or price (field variants supported: `price`, `variants.prices.centAmount`, `variants.prices.value.centAmount`).
- Defaults: sort by name asc, limit/offset apply after filter/sort.

### Cart actions
- `addLineItem` (requires `sku`, `quantity > 0`), `changeLineItemQuantity` (requires `lineItemId`, `quantity > 0`).
- Totals recalc on each update; delete sets cart state to `deleted`.

### CSV importer
- `cmd/importer` auto-detects product vs category CSV and can import a directory (categories first).
- Projects are created automatically if missing.
- Category keys are normalized (trim `-type` / `-types`); parent is inferred from `orderHint` if missing.

### Dev/Infra
- Docker Compose services: `db`, `db-test`, `migrate`, `api`, `api-dev` (air), `dev`, `pgadmin`.
- `./devenv` shells into the `dev` container (expects `docker compose --profile dev up -d dev`).
- `make test` brings up `db-test` and runs `go test ./...` inside `dev`.

### Known gaps
- Orders, inventory, checkout, discounts (beyond the static product-discounts list).
- `POST /oauth/token` is a stub; no refresh-token exchange.
- Product list responses are raw arrays (not full CT list objects).
