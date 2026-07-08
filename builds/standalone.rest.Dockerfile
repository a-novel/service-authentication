# This image exposes our app as a rest api server.
#
# It requires a database instance to run properly. The instance may not be patched.
#
# This image will make sure all patches are applied before starting the server. It is a larger
# version of the base rest image, suited for local development rather than full scale production.
FROM docker.io/library/golang:1.26.5-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY ./cmd/rest ./cmd/rest
COPY ./cmd/migrations ./cmd/migrations
COPY ./cmd/init ./cmd/init
COPY ./internal/handlers ./internal/handlers
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/core ./internal/core
COPY ./internal/models ./internal/models
COPY ./internal/config ./internal/config
COPY ./pkg ./pkg

# One RUN so the three binaries share a single warm module + build cache.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /rest ./cmd/rest/ && \
    go build -ldflags="-s -w" -trimpath -o /migrations ./cmd/migrations/ && \
    go build -ldflags="-s -w" -trimpath -o /init ./cmd/init/

FROM docker.io/library/alpine:3.24.1

WORKDIR /

COPY --from=builder /rest /rest
COPY --from=builder /migrations /migrations
COPY --from=builder /init /init

# Alpine ships BusyBox wget — no extra package needed for the healthcheck.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD wget -qO /dev/null http://localhost:8080/ping || exit 1

# Make sure the executable uses the default port.
ENV REST_PORT=8080

# Rest api port.
EXPOSE 8080

# Run patches before starting the server.
CMD ["sh", "-c", "/migrations && /init && /rest"]
