FROM golang:1.22-alpine AS build
WORKDIR /app

COPY go.mod ./
# Copying go.sum if present will allow cached downloads; optional now.
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go mod download || true

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go build -o /bin/api ./cmd/api

FROM alpine:3.20
WORKDIR /srv
COPY --from=build /bin/api /srv/api

ENV HTTP_ADDR=:8080
ENV DB_DSN=postgres://commerce:commerce@db:5432/commerce?sslmode=disable
ENV SHUTDOWN_TIMEOUT_SECONDS=10

EXPOSE 8080

ENTRYPOINT ["/srv/api"]
