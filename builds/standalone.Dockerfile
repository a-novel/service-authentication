# This image exposes our app as a rest api server.
#
# It requires a database instance to run properly. The instance may not be patched.
#
# This image will make sure all patches are applied before starting the server. It is a larger
# version of the base rest image, suited for local development rather than full scale production.
FROM docker.io/library/golang:1.25.5-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/rest" "./cmd/rest"
COPY "./cmd/migrations" "./cmd/migrations"
COPY "./cmd/init" "./cmd/init"
COPY ./internal/handlers ./internal/handlers
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/services ./internal/services
COPY ./internal/models ./internal/models
COPY ./internal/config ./internal/config
COPY ./pkg ./pkg

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /api cmd/rest/main.go
RUN go build -o /migrations cmd/migrations/main.go
RUN go build -o /init cmd/init/main.go

FROM docker.io/library/alpine:3.22.2

WORKDIR /

COPY --from=builder /api /api
COPY --from=builder /migrations /migrations
COPY --from=builder /init /init

# ======================================================================================================================
# Healthcheck.
# ======================================================================================================================
RUN apk --update add curl

HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD curl -f http://localhost:8080/ping || exit 1

# ======================================================================================================================
# Finish setup.
# ======================================================================================================================
# Make sure the executable uses the default port.
ENV PORT=8080

# Rest api port.
EXPOSE 8080

# Run patches before starting the server.
CMD ["sh", "-c", "/migrations && /init && /api"]
