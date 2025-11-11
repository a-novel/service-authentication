FROM docker.io/library/golang:1.25.4-alpine AS builder

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
ENV PORT=8080

EXPOSE 8080

# Make sure the migrations are run before the API starts.
CMD ["sh", "-c", "/migrations && /init && /api"]