FROM golang:alpine AS builder

WORKDIR /app

COPY . .

RUN go build -o /server cmd/server/main.go

FROM alpine:latest

WORKDIR /

COPY --from=builder /server /server

ENV PORT=8080

ENV HOST="0.0.0.0"

EXPOSE $PORT

# Run
CMD ["/server"]
