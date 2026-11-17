# Runs the REST server. Requires a database with migrations already applied.
FROM docker.io/library/golang:1.27.4-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY ./cmd ./cmd
COPY ./internal ./internal
COPY ./pkg/go ./pkg/go

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /rest ./cmd/rest/

FROM docker.io/library/alpine:3.24.4

COPY --from=builder /rest /rest

# The healthcheck uses the wget of Alpine's BusyBox.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD ["wget", "-qO", "/dev/null", "http://localhost:8080/v2/ping"]

ENV REST_PORT=8080

EXPOSE 8080

CMD ["/rest"]
