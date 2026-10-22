# Assemble signed Wolfi packages with a versioned upstream builder.
FROM docker.io/library/alpine:3.24.3 AS packages
SHELL ["/bin/ash", "-eo", "pipefail", "-c"]
# jq only reads the locks in this stage and never reaches the image, so its version can't change it.
# hadolint ignore=DL3018
RUN apk add --no-cache jq
# One pinned checksum list verifies the apko release archive of every architecture.
ARG TARGETARCH
ARG APKO_VERSION=1.4.12
ARG APKO_CHECKSUMS_SHA256=d53e2c729a37dbbbd99fe50c11b8362a7abe491e45b9763e0f603f200d59a637
WORKDIR /tmp
RUN archive="apko_${APKO_VERSION}_linux_${TARGETARCH}" \
    && wget -q "https://github.com/chainguard-dev/apko/releases/download/v${APKO_VERSION}/checksums.txt" \
        "https://github.com/chainguard-dev/apko/releases/download/v${APKO_VERSION}/$archive.tar.gz" \
    && echo "${APKO_CHECKSUMS_SHA256}  checksums.txt" | sha256sum -c - \
    && grep " $archive.tar.gz\$" checksums.txt | sha256sum -c - \
    && tar -xzf "$archive.tar.gz" -C /usr/local/bin --strip-components=1 "$archive/apko"
COPY ./builds/database.apko.yaml ./builds/database.apko.lock.json \
    ./builds/database.builder.apko.yaml ./builds/database.builder.apko.lock.json /apko/
# Install exactly the locked packages, and fail when a config resolves to a package its lock lacks.
RUN for config in database database.builder; do \
        lock="/apko/$config.apko.lock.json"; \
        apko build-minirootfs "/apko/$config.apko.yaml" "/$config.tar" --build-date 1970-01-01T00:00:00Z \
            --package-append "$(jq -r '[.contents.packages[] | "\(.name)=\(.version)"] | join(",")' "$lock")"; \
        mkdir "/$config" && tar -xf "/$config.tar" -C "/$config" --exclude=dev; \
        installed="$(awk '/^P:/ { name = substr($0, 3) } /^V:/ { print name "=" substr($0, 3) }' \
            "/$config/lib/apk/db/installed" | sort)"; \
        if [ "$installed" != "$(jq -r '.contents.packages[] | "\(.name)=\(.version)"' "$lock" | sort)" ]; then \
            echo "$lock is out of date: run apko lock on $config.apko.yaml" >&2; exit 1; \
        fi; \
    done
ARG PGBACKREST_VERSION=2.60.0
ARG PGBACKREST_SHA256=6d3ee1b5defdadc85228969b4be804096f5e65e54707bb95c7c4c4ce41f5d007
RUN wget -q -O /pgbackrest.tar.gz "https://github.com/pgbackrest/pgbackrest/releases/download/release/${PGBACKREST_VERSION}/pgbackrest-${PGBACKREST_VERSION}.tar.gz" \
    && echo "${PGBACKREST_SHA256}  /pgbackrest.tar.gz" | sha256sum -c -

# Wolfi supplies PostgreSQL; only pgBackRest needs its upstream source build.
FROM scratch AS backup-builder
COPY --from=packages /database.builder/ /
COPY --from=packages /pgbackrest.tar.gz /tmp/pgbackrest.tar.gz
RUN mkdir /tmp/pgbackrest \
    && tar -xzf /tmp/pgbackrest.tar.gz -C /tmp/pgbackrest --strip-components=1 \
    && meson setup /tmp/build /tmp/pgbackrest --buildtype=release \
    && ninja -C /tmp/build -j 2 \
    && meson test -C /tmp/build --print-errorlogs

FROM scratch
COPY --from=packages /database/ /
COPY --from=backup-builder /tmp/build/src/pgbackrest /usr/bin/pgbackrest
ENV PATH=/usr/libexec/postgresql18:/usr/local/bin:/usr/bin:/bin \
    PG_MAJOR=18 PGDATA=/var/lib/postgresql/18/docker LANG=en_US.utf8 \
    POSTGRES_INITDB_ARGS=--auth=scram-sha-256
COPY ./builds/database.sql /docker-entrypoint-initdb.d/init.sql
# Preserve the service initialization and recovery paths around Wolfi's layout.
RUN mkdir -p /usr/lib/postgresql/18 /usr/local/bin /var/lib/postgres \
    && ln -s /usr/libexec/postgresql18 /usr/lib/postgresql/18/bin \
    && ln -s /usr/lib/postgresql18 /usr/lib/postgresql/18/lib \
    && ln -s /usr/libexec/postgresql18/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh \
    && ln -s /docker-entrypoint-initdb.d /var/lib/postgres/initdb \
    && test "$(gosu postgres id -u)" = 999 \
    && gosu postgres pgbackrest version \
    && gosu postgres openssl crl2pkcs7 -nocrl -certfile /etc/ssl/certs/ca-certificates.crt -out /dev/null
EXPOSE 5432
STOPSIGNAL SIGINT
# The probe names the configured role and database: a role or database that does not exist makes
# the server log a failed connection on every probe.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD ["sh", "-c", "PGUSER=$POSTGRES_USER PGDATABASE=$POSTGRES_DB pg_isready"]
COPY --chmod=755 ./builds/database.entrypoint.sh /usr/local/bin/database.entrypoint.sh
ENTRYPOINT ["/usr/local/bin/database.entrypoint.sh"]
CMD ["postgres"]
