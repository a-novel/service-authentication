FROM golang:alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o /api cmd/api/main.go

FROM alpine:latest

WORKDIR /

COPY --from=builder /api /api

ENV HOST="0.0.0.0"

ENV PORT=8080

EXPOSE 8080

# Run
CMD ["/api"]
