# This image runs a job that will create / update a default super-admin user.
#
# It requires a patched database instance to run properly.
FROM docker.io/library/golang:1.26.4-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY ./cmd/init ./cmd/init
COPY ./internal/config ./internal/config
COPY ./internal/dao ./internal/dao
COPY ./internal/core ./internal/core
COPY ./internal/models/mails ./internal/models/mails
COPY ./internal/lib ./internal/lib

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /init ./cmd/init/

FROM docker.io/library/alpine:3.24.1

WORKDIR /

COPY --from=builder /init /init

CMD ["/init"]
