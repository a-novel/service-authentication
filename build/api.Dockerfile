FROM docker.io/library/golang:1.25.2-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/api" "./cmd/api"
COPY ./internal/api ./internal/api
COPY ./internal/dao ./internal/dao
COPY ./internal/lib ./internal/lib
COPY ./internal/services ./internal/services
COPY ./pkg ./pkg
COPY ./models ./models

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /api cmd/api/main.go

FROM docker.io/library/alpine:3.22.2

WORKDIR /

COPY --from=builder /api /api

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
CMD ["/api"]