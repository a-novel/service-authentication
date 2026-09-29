# This is a custom postgres image that comes with pre-loaded extensions. It allows us to customize
# our instance at build time.
#
# Note: this image does not run the migrations from the main image, make sure to call the appropriate
# patch for this.
FROM docker.io/library/golang:1.27.1-alpine AS gosu-builder

ENV CGO_ENABLED=0

# Build the upstream privilege helper with the supported compiler.
RUN GOBIN=/out go install -trimpath -ldflags="-s -w" github.com/tianon/gosu@1.19

FROM docker.io/library/postgres:18.6-trixie

COPY --from=gosu-builder /out/gosu /usr/local/bin/gosu

ENV POSTGRES_INITDB_ARGS=--auth=scram-sha-256

# Keep backup tooling with the database; archiving and scheduling remain opt-in.
# Preserve the base image's PostgreSQL binaries when installing packages.
RUN sha256sum /usr/lib/postgresql/18/bin/postgres /usr/lib/postgresql/18/lib/uuid-ossp.so > /tmp/database.sha256 \
    && apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
        ca-certificates=20250419 \
        pgbackrest=2.59.1-1.pgdg13+1 \
    && apt-get purge -y --auto-remove gnupg gnupg-l10n gpg gpg-agent gpgconf gpgsm dirmngr \
    && sha256sum --check /tmp/database.sha256 \
    && test "$(gosu postgres id -u)" = "$(id -u postgres)" \
    && gosu postgres pgbackrest version \
    && gosu postgres test -s /etc/ssl/certs/ca-certificates.crt \
    && gosu postgres openssl crl2pkcs7 -nocrl -certfile /etc/ssl/certs/ca-certificates.crt -out /dev/null \
    && rm -rf /var/lib/apt/lists/* /tmp/database.sha256

# ======================================================================================================================
# Prepare extension scripts.
# ======================================================================================================================
# Custom entrypoint used to run postgres with extensions.
COPY ./builds/database.entrypoint.sh /usr/local/bin/database.entrypoint.sh
RUN chmod +x /usr/local/bin/database.entrypoint.sh

# Initial migration of the image, used to setup extensions within postgres.
COPY ./builds/database.sql /docker-entrypoint-initdb.d/init.sql

# ======================================================================================================================
# Finish setup.
# ======================================================================================================================
# Default postgres port.
EXPOSE 5432

# Postgres does not provide a healthcheck by default.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD ["pg_isready"]

# Use our entrypoint instead of the native one.
ENTRYPOINT ["/usr/local/bin/database.entrypoint.sh"]

# Restore the original command from the base image.
CMD ["postgres"]
