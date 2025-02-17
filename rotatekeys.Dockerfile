FROM golang:alpine AS builder

WORKDIR /app

COPY . .

RUN go mod download

RUN go build -o /job cmd/rotatekeys/main.go

FROM alpine:latest

WORKDIR /

COPY --from=builder /job /job

# Run
CMD ["/job"]
