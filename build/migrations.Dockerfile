FROM docker.io/library/golang:alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/migrations" "./cmd/migrations"
COPY ./migrations ./migrations
COPY "./pkg/cmd/postgres.config.default.go" "./pkg/cmd/postgres.config.default.go"

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /migrations cmd/migrations/main.go

FROM docker.io/library/alpine:latest

WORKDIR /

COPY --from=builder /migrations /migrations

# Make sure the migrations are run before the job starts.
CMD ["/migrations"]
