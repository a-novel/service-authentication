# This image runs a job that will apply the latest migrations to a database instance.
FROM docker.io/library/golang:1.26.5-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY ./cmd/migrations ./cmd/migrations
COPY ./internal/config ./internal/config
COPY ./internal/models/migrations ./internal/models/migrations

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /migrations ./cmd/migrations/

FROM docker.io/library/alpine:3.24.1

WORKDIR /

COPY --from=builder /migrations /migrations

# Applies the migrations to a linked database instance.
CMD ["/migrations"]
