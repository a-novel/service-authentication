# This is a custom postgres image that comes with pre-loaded extensions. It allows us to customize
# our instance at build time.
#
# Note: this image does not run the migrations from the main image, make sure to call the appropriate
# patch for this.
FROM docker.io/library/postgres:18.1

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
  CMD pg_isready || exit 1

# Use our entrypoint instead of the native one.
ENTRYPOINT ["/usr/local/bin/database.entrypoint.sh"]

# Restore the original command from the base image.
CMD ["postgres"]
