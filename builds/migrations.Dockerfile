# This image runs a job that will apply the latest migrations to a database instance.
FROM docker.io/library/golang:1.25.4-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/migrations" "./cmd/migrations"
COPY ./internal/config ./internal/config
COPY ./internal/models/migrations ./internal/models/migrations

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /migrations cmd/migrations/main.go

FROM docker.io/library/alpine:3.22.2

WORKDIR /

COPY --from=builder /migrations /migrations

# Applies the migrations to a linked database instance.
CMD ["/migrations"]
