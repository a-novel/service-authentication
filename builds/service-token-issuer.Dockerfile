# This image stands in for a Kubernetes cluster's service-account issuer in local stacks and tests,
# so a stack can turn on the REST server's service-token check. It builds golib's command at the
# version go.mod requires, the one the server verifies tokens with. It signs a token for anyone who
# asks, so only a local stack may trust it.
FROM docker.io/library/golang:1.27.3-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

# The command lives in a dependency, so go.mod and go.sum are the whole build context.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /service-token-issuer github.com/a-novel-kit/golib/cmd/service-token-issuer

FROM docker.io/library/alpine:3.24.3

COPY --from=builder /service-token-issuer /service-token-issuer

# Alpine ships BusyBox wget, so the health check needs no extra package.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD ["wget", "-qO", "/dev/null", "http://localhost:8080/.well-known/openid-configuration"]

EXPOSE 8080

CMD ["/service-token-issuer"]
