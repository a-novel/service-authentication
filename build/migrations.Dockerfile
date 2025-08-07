FROM docker.io/library/golang:1.24.6-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/migrations" "./cmd/migrations"
COPY ./migrations ./migrations
COPY ./models ./models

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /migrations cmd/migrations/main.go

FROM docker.io/library/alpine:3.22.1

WORKDIR /

COPY --from=builder /migrations /migrations

# Make sure the migrations are run before the job starts.
CMD ["/migrations"]
