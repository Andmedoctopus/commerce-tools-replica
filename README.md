# commercetools-replica

Partial commercetools-compatible API (Go + Postgres), runnable via Docker Compose. Implements customers/auth, products, categories, carts, and search with commercetools-shaped responses where possible.


This project is an experiment where I try to avoid editing any code and do it the project with agent. The project is completely "vibe coded".

Agent: `gpt-5.2-codex`

## Prerequisites
- Docker + Docker Compose
- `./devenv` script available (uses the `dev` container)

## Quick start
1) Bring up infra (db, db-test, dev): `docker compose --profile dev up -d db db-test dev`
2) Run migrations: `./devenv go run ./cmd/migrate`
   - The `migrate` service runs `go run ./cmd/migrate` inside the dev image with your repo bind-mounted.
3) Import data:
   - Place commercetools exports under `imports/<projectKey>/` (e.g. `imports/petal_pot/Categories_...csv`, `imports/petal_pot/Products_...csv`)
   - The importer auto-detects file type by columns and will import every CSV in the project folder:
     ```
     ./devenv go run ./cmd/importer -project petal_pot
     ```
     (Pass `-path` to target a specific file or directory.)
4) Run API (dev hot reload on 8081): `docker compose up -d api-dev`
   - CT-style paths: `/:projectKey/...`
   - Health: `/healthz`, `/readyz`.

## API coverage
- Auth: `POST /oauth/:projectKey/customers/token` (password grant, form-encoded), `POST /oauth/:projectKey/anonymous/token` (client_credentials), `POST /oauth/token` (stub).
- Customers: `POST /:projectKey/me/signup`, `POST /:projectKey/me/login` (returns customer + active cart, no tokens), `GET /:projectKey/me` (bearer token).
- Products: `GET /:projectKey/products`, `GET /:projectKey/products/:id`, `POST /:projectKey/products/search` (price range + category filter, name/price sort).
- Categories: `GET /:projectKey/categories` (limit/offset).
- Carts: `POST /:projectKey/carts`, `GET /:projectKey/carts/:id` (raw cart shape), `POST /:projectKey/me/carts`, `POST /:projectKey/me/carts/:id` (actions: addLineItem, changeLineItemQuantity), `DELETE /:projectKey/me/carts/:id`, `GET /:projectKey/me/active-cart`.
- Product discounts: `GET /:projectKey/product-discounts` (static demo list).

Example payloads live in `req-example/` and `res-example/`.

## CSV expectations
- Product export: commercetools product CSV with `key`, `name.en`, `variants.sku`, `variants.prices.value.centAmount`, `variants.prices.value.currencyCode`. Images are read from `variants.images.url`. Categories come from `categories` or `productType.key` (normalized, `-types` stripped).
- Category export: CSV with columns like `key,name.en,slug.en,parent.key,orderHint` plus optional `description.en`, `metaTitle.en`, `metaDescription.en`. Missing key falls back to slug; name falls back to title-cased key. Parent is inferred from `orderHint` if `parent.key` is empty.

## Tests
Run inside dev container: `./devenv go test ./...`

## Notes
- Categories store parent key and build ancestors on read; no full tree materialization.
- `/me/*` endpoints require bearer tokens from `/oauth/:projectKey/...` token routes.
- CORS is open to localhost/127.0.0.1 for dev use.
