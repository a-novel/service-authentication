FROM docker.io/library/golang:1.25.0-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/api" "./cmd/api"
COPY "./cmd/migrations" "./cmd/migrations"
COPY ./internal/api ./internal/api
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/services ./internal/services
COPY ./migrations ./migrations
COPY ./pkg ./pkg
COPY ./models ./models

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /api cmd/api/main.go
RUN go build -o /migrations cmd/migrations/main.go

FROM docker.io/library/alpine:3.22.1

WORKDIR /

COPY --from=builder /api /api
COPY --from=builder /migrations /migrations

# ======================================================================================================================
# Healthcheck.
# ======================================================================================================================
RUN apk --update add curl

HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD curl -f http://localhost:8080/v1/healthcheck || exit 1

# ======================================================================================================================
# Finish setup.
# ======================================================================================================================
ENV PORT=8080

EXPOSE 8080

# Make sure the migrations are run before the API starts.
CMD ["sh", "-c", "/migrations && /api"]