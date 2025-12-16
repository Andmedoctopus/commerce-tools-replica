# commercetools-replica

Partial commercetools-compatible API (Go + Postgres), runnable via Docker Compose. Supports products, carts, and categories with commercetools-shaped responses.

## Prerequisites
- Docker + Docker Compose
- `./devenv` script available (uses the `dev` container)

## Quick start
1) Bring up infra (db, dev): `docker compose up -d db db-test dev`
2) Run migrations: `./devenv go run ./cmd/migrate`
3) Import data:
   - Products export (e.g. `imports/Products_Export_09-12-25_19-36.csv`)
   - Categories export (e.g. `imports/Categories_Export_14-12-25_21-34.csv`)

   The importer auto-detects file type by columns:
   ```
   ./devenv go run ./cmd/importer -file imports/Categories_Export_14-12-25_21-34.csv -project petal_pot
   ./devenv go run ./cmd/importer -file imports/Products_Export_09-12-25_19-36.csv -project petal_pot
   ```
   (Creates the project if missing; applies migrations automatically.)

4) Run API (dev hot reload on 8081): `docker compose up -d api-dev`
   - CT-style paths: `/:projectKey/products`, `/:projectKey/products/{id}`, `/:projectKey/products/search`, `/:projectKey/categories`.
   - Health: `/healthz`, `/readyz`.

## CSV expectations
- Product export: commercetools product CSV with `variants.sku`, `variants.prices.value.centAmount`, `productType.key`, optional `categories`, images, etc. Categories are derived from `categories` or `productType.key` (normalized, `-types` stripped).
- Category export: CSV with columns like `key,name.en,slug.en,parent.key,orderHint`. Missing key falls back to slug; name falls back to title-cased key.

## Tests
Run inside dev container: `./devenv go test ./...`

## Notes
- Categories are stored flat (key/name/slug/orderHint/parentKey) based on available CSV data; hierarchy is not reconstructed beyond parentKey.
- CORS is open to localhost/127.0.0.1 for dev use.
