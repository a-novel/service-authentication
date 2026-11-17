# Runs the maintenance job: one trusted, parameterized authentication operation. An operation that
# touches persisted state requires a migrated database. A registration invitation also requires the
# platform URL and SMTP configuration of the REST image.
FROM docker.io/library/golang:1.27.4-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /maintenance ./cmd/maintenance/

FROM docker.io/library/alpine:3.24.4

COPY --from=builder /maintenance /maintenance

ENTRYPOINT ["/maintenance"]
CMD ["--help"]
