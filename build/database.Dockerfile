FROM docker.io/library/postgres:17

# ======================================================================================================================
# Prepare extension scripts.
# ======================================================================================================================
# Custom entrypoint used to run extensions at runtime.
# We copy it to the same directory as the original entrypoint from the base image. Thus, our entrypoint can
# reference the original.
COPY ./build/database.entrypoint.sh /usr/local/bin/database.entrypoint.sh
RUN chmod +x /usr/local/bin/database.entrypoint.sh

COPY ./build/database.sql /docker-entrypoint-initdb.d/init.sql

# ======================================================================================================================
# Finish setup.
# ======================================================================================================================
EXPOSE 5432

# Postgres does not provide a healthcheck by default.
HEALTHCHECK --interval=1s --timeout=5s --retries=10 --start-period=1s \
  CMD pg_isready || exit 1

ENTRYPOINT ["/usr/local/bin/database.entrypoint.sh"]

# Restore original command from the base image.
CMD ["postgres"]
