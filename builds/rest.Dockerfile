# This image exposes our app as a rest api server.
#
# It requires a patched database instance to run properly.
FROM docker.io/library/golang:1.25.6-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/rest" "./cmd/rest"
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

FROM docker.io/library/alpine:3.23.3

WORKDIR /

COPY --from=builder /api /api

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

CMD ["/api"]
