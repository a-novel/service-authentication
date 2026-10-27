# Authentication service

Identity and session manager for the A-Novel platform.

[![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/agorastoryverse)](https://twitter.com/agorastoryverse)
[![Discord](https://img.shields.io/discord/1315240114691248138?logo=discord)](https://discord.gg/rp4Qr8cA)

<hr />

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/a-novel/service-authentication)
![GitHub repo file or directory count](https://img.shields.io/github/directory-file-count/a-novel/service-authentication)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/a-novel/service-authentication)

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/a-novel/service-authentication/main.yaml)
[![codecov](https://codecov.io/gh/a-novel/service-authentication/graph/badge.svg)](https://codecov.io/gh/a-novel/service-authentication)

![Coverage graph](https://codecov.io/gh/a-novel/service-authentication/graphs/sunburst.svg)

## What it does

Authentication owns **user identities** and their **sessions**. An account is an email and a password, hashed with Argon2id. A client trades the credentials for a short-lived access token and a long-lived refresh token, then renews the pair without signing in again. Signing out revokes the refresh token. A caller without an account gets an anonymous access token, which cannot be renewed. Every account carries a role, and each role grants the permissions that services check per route.

Registration, an email change and a password reset each need a single-use **short code**, emailed to the user. A stolen session token alone cannot take over an account.

The service exposes one **public REST API** and holds no signing key. [JSON Keys](https://github.com/a-novel/service-json-keys) signs and verifies every token over its private gRPC API, on a network the two services share. The Go client ships an auth middleware that any service mounts to verify tokens and enforce permissions locally.

## Deploying

The service runs as published OCI images plus a PostgreSQL database. Accounts live in Postgres, so the REST server scales to many replicas behind a load balancer. Each replica counts rate limits in its own memory, so every replica adds its own budget. The optional invitation list lives in a private Google Sheet.

A running [JSON Keys service](https://github.com/a-novel/service-json-keys) is a hard dependency. Authentication reaches it over its private gRPC port, which carries sensitive key material: keep that link on an unexposed network.

> **OpenTofu modules are the planned canonical deployment path.** Until they land, deploy the images with any container orchestrator — the composition below is the reference for which images to run, how they wire together, and the environment they expect.

| Image                                     | Role                                                                                          |
| ----------------------------------------- | --------------------------------------------------------------------------------------------- |
| `service-authentication/rest`             | Public REST API. The long-running server.                                                     |
| `service-authentication/jobs/migrations`  | One-shot schema migration job; runs to completion before the server or maintenance.           |
| `service-authentication/jobs/maintenance` | Trusted, parameterized one-shot operations. Running it without an operation only prints help. |
| `service-authentication/database`         | PostgreSQL with pgBackRest on Wolfi. Any PostgreSQL works in its place.                       |

Pin every image to the same release tag, such as the [latest release](https://github.com/a-novel/service-authentication/releases/latest). A production deployment runs `database`, then `migrations` to completion, then any number of `rest` replicas. Maintenance runs on its own when an operator requests an operation. Provide `POSTGRES_USER`, `POSTGRES_PASSWORD` and `POSTGRES_DATABASE` through the orchestrator, and store the password as a secret.

```yaml
services:
  postgres-authentication:
    image: ghcr.io/a-novel/service-authentication/database:v2.18.4
    networks: [api]
    environment:
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_DB: "${POSTGRES_DATABASE}"
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - authentication-postgres-data:/var/lib/postgresql/
    # Podman ignores the image's own check, so the dependents' wait needs this one.
    healthcheck:
      test: ["CMD", "pg_isready"]
      interval: 1s
      timeout: 5s
      retries: 10

  migrations-authentication:
    image: ghcr.io/a-novel/service-authentication/jobs/migrations:v2.18.4
    depends_on:
      postgres-authentication: { condition: service_healthy }
    environment:
      POSTGRES_HOST: postgres-authentication
      POSTGRES_PORT: "5432"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
      POSTGRES_DATABASE: "${POSTGRES_DATABASE}"
      POSTGRES_TLS_ENABLED: "false"
    networks: [api]

  service-authentication:
    image: ghcr.io/a-novel/service-authentication/rest:v2.18.4
    ports: ["${SERVICE_AUTHENTICATION_REST_PORT}:8080"]
    depends_on:
      postgres-authentication: { condition: service_healthy }
      migrations-authentication: { condition: service_completed_successfully }
    environment:
      POSTGRES_HOST: postgres-authentication
      POSTGRES_PORT: "5432"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
      POSTGRES_DATABASE: "${POSTGRES_DATABASE}"
      POSTGRES_TLS_ENABLED: "false"
      SERVICE_JSON_KEYS_HOST: "<json-keys-host>"
      SERVICE_JSON_KEYS_PORT: "<json-keys-grpc-port>"
    networks: [api]

networks:
  api:

volumes:
  authentication-postgres-data:
```

The maintenance image takes an operation as command arguments. `account-role` gives an email address one exact role:

```text
maintenance account-role --email operator@example.com --role auth:admin --lang en
```

An existing account moves to the requested role. A missing account receives a registration link whose short code carries the role, and completing the registration creates the account with it. Repeating a request changes nothing, while a request for another role replaces the pending invitation. The roles are `auth:anon`, `auth:user`, `auth:admin` and `auth:superadmin`.

Without SMTP settings, emails go to a debug sender that prints them to stdout. The maintenance job waits for the invitation to go out, and fails when the sender reports an error.

`refresh-token-purge` deletes the revocations of refresh tokens that have expired, which no longer serve any check. Signing out adds one revocation each time, so run it on a schedule, daily for instance. It takes no parameters:

```text
maintenance refresh-token-purge
```

### Database image

The database image packages PostgreSQL and pgBackRest on Wolfi. The container build assembles its
[package manifest](./builds/database.apko.yaml) with apko, so Docker and Podman need no other host
tool. PostgreSQL keeps its standard `POSTGRES_*` initialization variables, the `uuid-ossp`
extension and its volume at `/var/lib/postgresql`. Backup scheduling and repository credentials
stay with the operator.

Moving from the Debian-based image takes a fresh volume. To keep existing data, restore a logical
dump into the new database and validate it before switching traffic. A Debian data directory and
its physical backups do not carry over, since the libc, collation and extensions differ. Keep the
previous image with its backups.

Wolfi's package repository rolls forward, so an old package manifest may stop building. Keep the
published service images for recovery.

### Configuration

The service reads every variable from the process environment, prefixed with
`SERVICE_AUTHENTICATION_ENV_PREFIX` when that variable is set.

| Name                     | Description                                                                                                                         | Images                                                                    |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| `POSTGRES_HOST`          | PostgreSQL hostname or IP address. **Required.**                                                                                    | `rest`<br/>`jobs/migrations`<br/>`jobs/maintenance`<br/>`standalone-rest` |
| `POSTGRES_PORT`          | PostgreSQL port. Defaults to `5432`.                                                                                                | `rest`<br/>`jobs/migrations`<br/>`jobs/maintenance`<br/>`standalone-rest` |
| `POSTGRES_USER`          | PostgreSQL login role. **Required.**                                                                                                | `rest`<br/>`jobs/migrations`<br/>`jobs/maintenance`<br/>`standalone-rest` |
| `POSTGRES_PASSWORD`      | PostgreSQL login password. **Required; inject it as a secret.**                                                                     | `rest`<br/>`jobs/migrations`<br/>`jobs/maintenance`<br/>`standalone-rest` |
| `POSTGRES_DATABASE`      | PostgreSQL database name. **Required.**                                                                                             | `rest`<br/>`jobs/migrations`<br/>`jobs/maintenance`<br/>`standalone-rest` |
| `POSTGRES_TLS_ENABLED`   | Encrypt the PostgreSQL connection. Defaults to `true`; disable only when another trusted boundary protects the database link.       | `rest`<br/>`jobs/migrations`<br/>`jobs/maintenance`<br/>`standalone-rest` |
| `SERVICE_JSON_KEYS_HOST` | Hostname of the [JSON Keys service](https://github.com/a-novel/service-json-keys), without scheme or port. Defaults to `localhost`. | `rest`<br/>`standalone-rest`                                              |
| `SERVICE_JSON_KEYS_PORT` | gRPC port of the JSON Keys service. Defaults to `8080`.                                                                             | `rest`<br/>`standalone-rest`                                              |

<details>
<summary>Optional configuration (client platform, SMTP, REST tuning, OpenTelemetry, Sentry)</summary>

**Client platform** (images `rest`, `jobs/maintenance`, `standalone-rest`). The links in outgoing emails point at the web platform that serves the account pages, such as [platform-studio](https://github.com/a-novel/platform-studio):

| Name                                | Description                      | Default                                     |
| ----------------------------------- | -------------------------------- | ------------------------------------------- |
| `PLATFORM_AUTH_URL`                 | Base URL of the client platform. |                                             |
| `PLATFORM_AUTH_URL_UPDATE_EMAIL`    | Email-validation page.           | `PLATFORM_AUTH_URL` + `/ext/email/validate` |
| `PLATFORM_AUTH_URL_UPDATE_PASSWORD` | Password-reset page.             | `PLATFORM_AUTH_URL` + `/ext/password/reset` |
| `PLATFORM_AUTH_URL_REGISTER`        | Register page.                   | `PLATFORM_AUTH_URL` + `/ext/account/create` |

**SMTP** (images `rest`, `jobs/maintenance`, `standalone-rest`). Without `SMTP_ADDR`, a debug sender prints emails to stdout. Emails carry short codes, so production needs a real server. Without `SMTP_SENDER_PASSWORD`, the service sends unauthenticated, to a relay that admits it by its IP address, and the connection must still upgrade with STARTTLS:

| Name                     | Description                                                                                                    | Default             |
| ------------------------ | -------------------------------------------------------------------------------------------------------------- | ------------------- |
| `SMTP_ADDR`              | SMTP server address (`domain:port`).                                                                           |                     |
| `SMTP_SENDER_NAME`       | Display name on outgoing emails.                                                                               |                     |
| `SMTP_SENDER_EMAIL`      | Sender address.                                                                                                |                     |
| `SMTP_USERNAME`          | SMTP authentication identity; treat as protected account metadata. Unused without a password.                  | `SMTP_SENDER_EMAIL` |
| `SMTP_SENDER_PASSWORD`   | SMTP authentication password. Sensitive — handle with care. Empty for an IP-only relay.                        |                     |
| `SMTP_SENDER_DOMAIN`     | Server name authentication is bound to; must match the host portion of `SMTP_ADDR`. Unused without a password. |                     |
| `SMTP_TIMEOUT`           | Send timeout.                                                                                                  | `20s`               |
| `SMTP_MAX_CONCURRENT`    | Maximum number of emails delivered at once; further sends wait for a free slot.                                | `16`                |
| `SMTP_FORCE_UNENCRYPTED` | **Never set in production.** Allows a connection without TLS, and plain credentials over it; dev only.         | `false`             |

**REST tuning** (images `rest`, `standalone-rest`):

| Name                          | Description                                                                                                                                  | Default                          |
| ----------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------- |
| `REST_PORT`                   | Port the server listens on.                                                                                                                  | `8080`                           |
| `REST_MAX_REQUEST_SIZE`       | Maximum request body size, in bytes.                                                                                                         | `16384` (16KiB)                  |
| `REST_CLIENT_IP_HEADER`       | Header a trusted proxy sets to the visitor IP. Set it only when every request comes through that proxy.                                      | TCP peer                         |
| `REST_LIVENESS_PATH`          | Ping path, which still answers during a planned downtime and skips the standard rate limit.                                                  | `/v2/ping`                       |
| `REST_RATE_LIMITS_ENABLED`    | Rate-limit each caller and email, as documented per route in the OpenAPI specification. Turn off only in test stacks.                        | `true`                           |
| `REST_RATE_LIMIT`             | Requests each visitor sends to each route, as `<requests>/<window>`.                                                                         | `5/1s`                           |
| `REST_RATE_LIMIT_STRICT`      | Anonymous tokens, invitation requests, and reset or email-change mails per caller.                                                           | `3/5m`                           |
| `REST_RATE_LIMIT_LOGIN_BURST` | Logins, sign-ups, password resets and changes, and email updates per caller over a short window.                                             | `10/1m`                          |
| `REST_RATE_LIMIT_LOGIN`       | The same per caller, and per email or account, over a longer window.                                                                         | `30/10m`                         |
| `REST_RATE_LIMIT_MAILBOX`     | Emails each address receives from one route.                                                                                                 | `3/1h`                           |
| `ARGON2_MAX_CONCURRENT`       | Password hashes running at once per instance; each takes 64 MiB. Hashing is CPU-bound, so more than one per CPU adds memory, not throughput. | CPU count (`GOMAXPROCS`)         |
| `REST_TIMEOUT_READ`           | Read timeout.                                                                                                                                | `15s`                            |
| `REST_TIMEOUT_READ_HEADER`    | Header read timeout.                                                                                                                         | `3s`                             |
| `REST_TIMEOUT_WRITE`          | Write timeout.                                                                                                                               | `30s`                            |
| `REST_TIMEOUT_IDLE`           | Idle keep-alive timeout.                                                                                                                     | `60s`                            |
| `REST_TIMEOUT_REQUEST`        | Per-request timeout.                                                                                                                         | `60s`                            |
| `REST_TIMEOUT_SHUTDOWN`       | Graceful shutdown budget. With the 4 seconds of telemetry flushes after it, it fits Cloud Run's 10 seconds before `SIGKILL`.                 | `6s`                             |
| `REST_CORS_ALLOWED_ORIGINS`   | CORS allowed origins.                                                                                                                        | `*`                              |
| `REST_CORS_ALLOWED_HEADERS`   | CORS allowed headers.                                                                                                                        | `*`                              |
| `REST_CORS_ALLOWED_METHODS`   | CORS allowed methods, comma-separated.                                                                                                       | `HEAD,GET,POST,PUT,PATCH,DELETE` |
| `REST_CORS_ALLOW_CREDENTIALS` | CORS allow-credentials flag.                                                                                                                 | `false`                          |
| `REST_CORS_MAX_AGE`           | CORS max-age, in seconds.                                                                                                                    | `3600`                           |

**Database connection pool** (server images). The limits apply **per process**. The database's `max_connections` has to cover every replica plus the migration job, and the stock `postgres` default is 100.

| Name                      | Description                               | Default |
| ------------------------- | ----------------------------------------- | ------- |
| `POSTGRES_MAX_OPEN_CONNS` | Maximum open connections to the database. | `20`    |
| `POSTGRES_MAX_IDLE_CONNS` | Maximum connections kept open while idle. | `20`    |

**Planned downtime** (images `rest`, `jobs/maintenance`, `standalone-rest`). From the start of a downtime until operators remove the variable, even past the announced end, the server answers `503` to everything but ping and starts without the database. Maintenance then does nothing. A downtime refusal from JSON Keys answers `503` whether or not this variable is set.

| Name             | Description                                         | Default |
| ---------------- | --------------------------------------------------- | ------- |
| `DOWNTIME_START` | When a planned downtime starts, in RFC 3339 format. |         |

**Logs and tracing** (images `rest`, `jobs/maintenance`, `standalone-rest`). OpenTelemetry exports to stdout, or over OTLP/HTTP once an endpoint is set:

| Name                             | Description                                                                                                      | Default                  |
| -------------------------------- | ---------------------------------------------------------------------------------------------------------------- | ------------------------ |
| `OTEL`                           | Enable OTel tracing; the variables below pick the exporter.                                                      | `false`                  |
| `GCLOUD_PROJECT_ID`              | Google Cloud project ID. When set, the JSON Keys connection authenticates with Google ID tokens.                 |                          |
| `OTEL_EXPORTER_OTLP_ENDPOINT`    | Base URL of the OTLP/HTTP backend. When set, traces and logs export there, and the servers write no request log. |                          |
| `OTEL_EXPORTER_OTLP_HEADERS`     | Headers each export sends, such as the backend's key. Inject it as a secret.                                     |                          |
| `OTEL_EXPORTER_OTLP_COMPRESSION` | `gzip` compresses each export.                                                                                   |                          |
| `OTEL_SERVICE_NAME`              | Service name every span and log record carries.                                                                  |                          |
| `APP_NAME`                       | Application name attached to traces and logs.                                                                    | `service-authentication` |

**Fault reporting** (images `rest`, `jobs/maintenance`, `standalone-rest`). With a DSN, the server reports each 5xx answer and each panic to Sentry, and maintenance reports a failed operation under its name. Without one, nothing is sent:

| Name                 | Description                                                                                                | Default |
| -------------------- | ---------------------------------------------------------------------------------------------------------- | ------- |
| `SENTRY_DSN`         | Client key of the service's Sentry project. Store it as a secret.                                          |         |
| `SENTRY_RELEASE`     | Deployed release, as `service-authentication@<version>`, the name release CI records. Required with a DSN. |         |
| `SENTRY_ENVIRONMENT` | Deployment name, such as `production`. Required with a DSN.                                                |         |

</details>

### Invitation list

Account creation is invitation-only, and the invitation list collects requests for an invitation.
`PUT /v2/waitlist` takes any session, anonymous included, and creates no account or short code. An
existing account or a repeated request gets a distinct conflict warning, and no email or row.
Creating the account removes its address from the list.

The integration stays off until the REST server has both `WAITLIST_URL` and `WAITLIST_SECRET`.
Maintenance jobs that read the list take the same configuration:

| Name                        | Purpose                                                                  | Default  |
| --------------------------- | ------------------------------------------------------------------------ | -------- |
| `WAITLIST_URL`              | Google Apps Script deployment URL ending in `/exec`.                     | Disabled |
| `WAITLIST_SECRET`           | Random signing key shared with the script, injected as a runtime secret. | Disabled |
| `WAITLIST_TIMEOUT`          | Timeout for one writer operation, including Google's response redirect.  | `10s`    |
| `WAITLIST_MAX_CONCURRENT`   | Waitlist requests waiting on Google at once, per instance.               | `4`      |
| `WAITLIST_MAX_REQUEST_SIZE` | Maximum waitlist request body size, in bytes.                            | `8192`   |

The [invitation list guide](./scripts/waitlist/README.md) installs the private sheet and covers its
operations. A Google outage blocks neither login nor registration.

### Shutting down cleanly

On `SIGTERM`, the server stops accepting requests, drains the open ones, then waits for the emails
already going out. Each email goes out on its own goroutine, so a send outlives the request that
started it.

The drain ends `REST_TIMEOUT_SHUTDOWN` after the signal, 6 seconds by default. The server then
flushes Sentry and OpenTelemetry for up to 2 seconds each, inside the 10 seconds that Cloud Run,
Docker and Podman grant before `SIGKILL`. A send still going out when the drain ends is cut and
reported to Sentry, and its user asks for another email.

## Using the client packages

Two clients ship with the service. [pkg.go.dev](https://pkg.go.dev/github.com/a-novel/service-authentication/v2) and the [API reference](https://a-novel.github.io/service-authentication) document their full surface.

- **Go** mounts an auth middleware in a backend service that gates routes on permissions. It verifies tokens through a JSON Keys client and never calls the authentication service.
- **JavaScript / TypeScript** calls the REST API, from a frontend or a Node service that drives the login, session and account flows.

### Go

```bash
go get github.com/a-novel/service-authentication/v2
```

```go
package main

import (
	"os"

	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/go-chi/chi/v5"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
)

// Each role grants permissions and inherits those of the roles it lists. The keys are
// the roles your tokens carry, such as the auth:* roles the service issues.
var myPermissions = serviceauthentication.Permissions{
	Roles: map[string]serviceauthentication.Role{
		"auth:user":  {Priority: 0, Permissions: []string{"post:read"}},
		"auth:admin": {Priority: 1, Inherits: []string{"auth:user"}, Permissions: []string{"post:write"}},
	},
}

func main() {
	jsonKeysClient, _ := servicejsonkeys.NewClient("service-json-keys:8080")
	verifier, _ := servicejsonkeys.NewClaimsVerifier[serviceauthentication.Claims](jsonKeysClient)
	logger := &loggingpresets.LogLocal{Out: os.Stdout}

	withAuth := serviceauthentication.NewAuthHandler(verifier, myPermissions, logger)
	router := chi.NewRouter()

	withAuth(router, "post:write").Get(...) // requires the post:write permission
	withAuth(router).Get(...)               // any caller, signed in or not
}
```

### JavaScript / TypeScript

GitHub Packages hosts the package and requires a personal access token with the `read:packages` scope, even for a public package ([why](https://github.com/orgs/community/discussions/23386#discussioncomment-3240193)). Add the registries to the `.npmrc` of your project or home directory:

```ini
@a-novel:registry=https://npm.pkg.github.com
@a-novel-kit:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${YOUR_PERSONAL_ACCESS_TOKEN}
```

```bash
pnpm add @a-novel/service-authentication-rest
```

```typescript
import { AuthenticationApi, tokenCreateAnon } from "@a-novel/service-authentication-rest";

const api = new AuthenticationApi("http://service-authentication:8080");

// Open an anonymous session, the entry point of the registration and login flows.
const token = await tokenCreateAnon(api);
```

Every call ships [Zod](https://github.com/colinhacks/zod) request and response schemas, and validates the response it receives.

## Running locally

For a throwaway instance without the development toolchain, the **`standalone-rest`** image bundles the server, the migrations and the maintenance executable in one container. It runs the migrations before the server on every boot, and runs maintenance only on request.

```yaml
services:
  postgres-authentication:
    image: ghcr.io/a-novel/service-authentication/database:v2.18.4
    networks: [api]
    environment:
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_DB: "${POSTGRES_DATABASE}"
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    # Podman ignores the image's own check, so the dependents' wait needs this one.
    healthcheck:
      test: ["CMD", "pg_isready"]
      interval: 1s
      timeout: 5s
      retries: 10

  service-authentication:
    image: ghcr.io/a-novel/service-authentication/standalone-rest:v2.18.4
    ports: ["${SERVICE_AUTHENTICATION_REST_PORT}:8080"]
    depends_on:
      postgres-authentication: { condition: service_healthy }
    environment:
      POSTGRES_HOST: postgres-authentication
      POSTGRES_PORT: "5432"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
      POSTGRES_DATABASE: "${POSTGRES_DATABASE}"
      POSTGRES_TLS_ENABLED: "false"
      SERVICE_JSON_KEYS_HOST: "<json-keys-host>"
      SERVICE_JSON_KEYS_PORT: "<json-keys-grpc-port>"
    networks: [api]

networks:
  api:
```

To work on the service itself, run it with the `a-novel` CLI as [CONTRIBUTING](./CONTRIBUTING.md) shows.

## Contributing

Platform setup and the day-to-day commands live in the [developer guide](https://a-novel-kit.github.io/stack/). Service-specific concepts and local interactions are in [CONTRIBUTING.md](./CONTRIBUTING.md).
