# This image exposes our app as a rest api server.
#
# It requires a patched database instance to run properly.
FROM docker.io/library/golang:1.26.5-alpine AS builder

# Static binary: no libc dependency, so it runs on a bare alpine base and could
# move to scratch/distroless later.
ENV CGO_ENABLED=0

WORKDIR /app

# Resolve modules off go.mod/go.sum alone, before the source, so the download
# layer survives any source-only edit. The cache mount persists the module cache
# across builds — locally this is what turns a cold re-pull into a no-op; in CI
# the gha layer cache covers cross-run reuse.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY ./cmd/rest ./cmd/rest
COPY ./internal/handlers ./internal/handlers
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/core ./internal/core
COPY ./internal/models ./internal/models
COPY ./internal/config ./internal/config
COPY ./pkg ./pkg

# Cache mounts persist the module + build cache across builds, so a rebuild after
# a source change recompiles only the changed packages instead of the whole tree.
# -ldflags="-s -w" strips the symbol table + DWARF (smaller binary); -trimpath
# drops absolute build paths for a reproducible build.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /rest ./cmd/rest/

FROM docker.io/library/alpine:3.24.1

WORKDIR /

COPY --from=builder /rest /rest

# Alpine ships BusyBox wget — no extra package needed for the healthcheck.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD wget -qO /dev/null http://localhost:8080/ping || exit 1

# Make sure the executable uses the default port.
ENV REST_PORT=8080

# Rest api port.
EXPOSE 8080

CMD ["/rest"]
