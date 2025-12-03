# This image runs a job that will create / update a default super-admin user.
#
# It requires a patched database instance to run properly.
FROM docker.io/library/golang:1.25.5-alpine AS builder

WORKDIR /app

# ======================================================================================================================
# Copy build files.
# ======================================================================================================================
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
COPY "./cmd/init" "./cmd/init"
COPY ./internal/config ./internal/config
COPY ./internal/dao ./internal/dao
COPY ./internal/services ./internal/services
COPY ./internal/models/mails ./internal/models/mails
COPY ./internal/lib ./internal/lib

RUN go mod download

# ======================================================================================================================
# Build executables.
# ======================================================================================================================
RUN go build -o /init cmd/init/main.go

FROM docker.io/library/alpine:3.23.0

WORKDIR /

COPY --from=builder /init /init

CMD ["/init"]
