# Runs the migrations, then the REST server, in one image for local development and tests. It also
# bundles the maintenance command, which startup never runs. A production deployment runs the
# rest.Dockerfile image after the migrations job instead.
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
    go build -ldflags="-s -w" -trimpath -o /rest ./cmd/rest/ && \
    go build -ldflags="-s -w" -trimpath -o /migrations ./cmd/migrations/ && \
    go build -ldflags="-s -w" -trimpath -o /maintenance ./cmd/maintenance/

FROM docker.io/library/alpine:3.24.4

COPY --from=builder /rest /rest
COPY --from=builder /migrations /migrations
COPY --from=builder /maintenance /maintenance

# The healthcheck uses the wget of Alpine's BusyBox.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD ["wget", "-qO", "/dev/null", "http://localhost:8080/v2/ping"]

ENV REST_PORT=8080
# A standalone stack is local or a test: it runs without service tokens unless given an issuer.
ENV SERVICE_ISSUER_OPTIONAL=true

EXPOSE 8080

CMD ["sh", "-c", "/migrations && exec /rest"]
