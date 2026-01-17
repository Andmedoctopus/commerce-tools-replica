FROM golang:1.25-alpine AS build
WORKDIR /app

COPY go.mod ./
# Copying go.sum if present will allow cached downloads; optional now.
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    mkdir /app/bin && \
    go build -o /app/bin/api ./cmd/api && \
    go build -o /app/bin/migrate ./cmd/migrate && \
    go build -o /app/bin/importer ./cmd/importer

FROM golang:1.25-alpine AS dev
WORKDIR /workspace
RUN --mount=type=cache,target=/var/cache/apk \
    apk add --no-cache git bash build-base
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    GOBIN=/usr/local/bin go install github.com/air-verse/air@v1.63.4

FROM alpine:3.20 AS prod
WORKDIR /srv
COPY --from=build /app/bin/* /srv/

ENV HTTP_ADDR=:8080
ENV DB_DSN=postgres://commerce:commerce@db:5432/commerce?sslmode=disable
ENV SHUTDOWN_TIMEOUT_SECONDS=10

EXPOSE 8080

ENTRYPOINT ["/srv/api"]
