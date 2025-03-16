FROM golang:alpine AS builder

WORKDIR /app

COPY ../api ./api
COPY ../cmd/api ./cmd/api
COPY ../config ./config
COPY ../internal/dao ./internal/dao
COPY ../internal/lib ./internal/lib
COPY ../internal/services ./internal/services
COPY ../migrations ./migrations
COPY ../models ./models
COPY ../go.mod ./go.mod
COPY ../go.sum ./go.sum

RUN go mod download

RUN go build -o /api cmd/api/main.go

FROM gcr.io/distroless/base:latest

WORKDIR /

COPY --from=builder /api /api

ENV HOST="0.0.0.0"

ENV PORT=8080

EXPOSE 8080

# Run
CMD ["/api"]
