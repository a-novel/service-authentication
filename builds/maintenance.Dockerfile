# This image runs trusted, parameterized authentication maintenance operations.
#
# Operations that touch persisted state require a migrated database. Registration
# invitations also require the platform URL and SMTP configuration used by the REST image.
FROM docker.io/library/golang:1.27.1-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY ./cmd/maintenance ./cmd/maintenance
COPY ./internal/config ./internal/config
COPY ./internal/dao ./internal/dao
COPY ./internal/core ./internal/core
COPY ./internal/models/mails ./internal/models/mails
COPY ./internal/lib ./internal/lib

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /maintenance ./cmd/maintenance/

FROM docker.io/library/alpine:3.24.2

WORKDIR /

COPY --from=builder /maintenance /maintenance

ENTRYPOINT ["/maintenance"]
CMD ["--help"]
