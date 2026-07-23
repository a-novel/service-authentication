# Authentication service

Identity and session manager for the A-Novel platform: it owns user credentials, issues and refreshes the access/refresh token pair, and ships a Go middleware so any service can gate its routes on roles and permissions.

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

Authentication owns **user identities** — email/password credentials, hashed with Argon2id — and the **token lifecycle**. Clients trade credentials for a short-lived access token and a long-lived refresh token, then refresh the pair without re-authenticating; callers with no account get an anonymous, access-only token that cannot be refreshed. Every account carries a role, and each role maps to a set of permissions that downstream services enforce per route.

Identity changes — registration, email change, password reset — are gated by single-use **short codes** emailed to the user, so a stolen session token alone can't take over an account.

It exposes one **public REST API** and signs nothing itself: signing and verification go to [JSON Keys](https://github.com/a-novel/service-json-keys) over that service's private gRPC API, so the two share a secure, unexposed network. The Go client also ships an auth middleware any service can mount to verify tokens and enforce permissions locally.

## Deploying

The service runs as published OCI images plus a PostgreSQL database. The REST server is stateless, so it scales to as many replicas as you need behind a load balancer; all state lives in Postgres. A running [JSON Keys service](https://github.com/a-novel/service-json-keys) is a hard dependency — authentication reaches it over its private gRPC port, and the two share sensitive key material, so keep that link on an unexposed network.

> **OpenTofu modules are the planned canonical deployment path.** Until they land, deploy the images with any container orchestrator — the composition below is the reference for which images to run, how they wire together, and the environment they expect.

| Image                                    | Role                                                                                 |
| ---------------------------------------- | ------------------------------------------------------------------------------------ |
| `service-authentication/rest`            | Public REST API. The long-running server.                                            |
| `service-authentication/jobs/migrations` | One-shot schema migration job; runs to completion before `init` and the server.      |
| `service-authentication/jobs/init`       | One-shot bootstrap job; provisions the super-admin from `SUPER_ADMIN_*`. Idempotent. |
| `service-authentication/database`        | Pre-tuned PostgreSQL image — or bring your own Postgres.                             |

Pin every image to the same release tag — see the [latest release](https://github.com/a-novel/service-authentication/releases/latest). A production deployment runs `database`, then `migrations` to completion, then `init` to completion, then any number of `rest` replicas:

```yaml
services:
  postgres-authentication:
    image: ghcr.io/a-novel/service-authentication/database:v2.4.5
    networks: [api]
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - authentication-postgres-data:/var/lib/postgresql/

  migrations-authentication:
    image: ghcr.io/a-novel/service-authentication/jobs/migrations:v2.4.3
    depends_on:
      postgres-authentication: { condition: service_healthy }
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-authentication:5432/postgres?sslmode=disable"
    networks: [api]

  # Optional: seeds the initial super-admin user. Pass the credentials securely.
  init-authentication:
    image: ghcr.io/a-novel/service-authentication/jobs/init:v2.4.3
    depends_on:
      postgres-authentication: { condition: service_healthy }
      migrations-authentication: { condition: service_completed_successfully }
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-authentication:5432/postgres?sslmode=disable"
      SUPER_ADMIN_EMAIL: "<super-admin-email>"
      SUPER_ADMIN_PASSWORD: "<super-admin-password>"
    networks: [api]

  service-authentication:
    image: ghcr.io/a-novel/service-authentication/rest:v2.4.5
    ports: ["${SERVICE_AUTHENTICATION_REST_PORT}:8080"]
    depends_on:
      postgres-authentication: { condition: service_healthy }
      migrations-authentication: { condition: service_completed_successfully }
      init-authentication: { condition: service_completed_successfully }
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-authentication:5432/postgres?sslmode=disable"
      SERVICE_JSON_KEYS_HOST: "<json-keys-host>"
      SERVICE_JSON_KEYS_PORT: "<json-keys-grpc-port>"
    networks: [api]

networks:
  api:

volumes:
  authentication-postgres-data:
```

The `init` job is idempotent — leave `SUPER_ADMIN_*` unset and it exits without touching the database, so it is safe to keep in every deployment. The server is wired to wait on it (`depends_on`), so if you drop the `init` service entirely, remove that dependency from `service-authentication` too or it won't start. Email-bearing flows (registration, password reset, email change) fall back to a debug sender that prints to stdout unless you configure SMTP — see the optional configuration below.

### Configuration

Every variable is read from the process environment.

| Name                     | Description                                                                                                                     | Images                                                             |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| `POSTGRES_DSN`           | PostgreSQL connection string. **Required.**                                                                                     | `rest`<br/>`jobs/migrations`<br/>`jobs/init`<br/>`standalone-rest` |
| `SERVICE_JSON_KEYS_HOST` | Hostname of the [JSON Keys service](https://github.com/a-novel/service-json-keys) (no scheme/port). **Required** on the server. | `rest`<br/>`standalone-rest`                                       |
| `SERVICE_JSON_KEYS_PORT` | gRPC port of the JSON Keys service. **Required** on the server.                                                                 | `rest`<br/>`standalone-rest`                                       |
| `SUPER_ADMIN_EMAIL`      | Email of the super-admin to provision. The bootstrap is skipped if unset.                                                       | `jobs/init`<br/>`standalone-rest`                                  |
| `SUPER_ADMIN_PASSWORD`   | Plaintext password for the super-admin. Pass it securely. The bootstrap is skipped if unset.                                    | `jobs/init`<br/>`standalone-rest`                                  |

<details>
<summary>Optional configuration (client platform, SMTP, REST tuning, OpenTelemetry)</summary>

**Client platform** — optional; only used to build links in outgoing emails. Point it at a running [authentication platform](https://github.com/a-novel/platform-authentication) (images `rest`, `standalone-rest`):

| Name                                | Description                      | Default                                     |
| ----------------------------------- | -------------------------------- | ------------------------------------------- |
| `PLATFORM_AUTH_URL`                 | Base URL of the client platform. |                                             |
| `PLATFORM_AUTH_URL_UPDATE_EMAIL`    | Email-validation page.           | `PLATFORM_AUTH_URL` + `/ext/email/validate` |
| `PLATFORM_AUTH_URL_UPDATE_PASSWORD` | Password-reset page.             | `PLATFORM_AUTH_URL` + `/ext/password/reset` |
| `PLATFORM_AUTH_URL_REGISTER`        | Register page.                   | `PLATFORM_AUTH_URL` + `/ext/account/create` |

**SMTP** — without these, emails are printed to stdout by a debug sender (dev only; set a real server in production, since emails carry short codes) (images `rest`, `standalone-rest`):

| Name                     | Description                                                                                  | Default |
| ------------------------ | -------------------------------------------------------------------------------------------- | ------- |
| `SMTP_ADDR`              | SMTP server address (`domain:port`).                                                         |         |
| `SMTP_SENDER_NAME`       | Display name on outgoing emails.                                                             |         |
| `SMTP_SENDER_EMAIL`      | Sender address.                                                                              |         |
| `SMTP_SENDER_PASSWORD`   | Sender account password. Sensitive — handle with care.                                       |         |
| `SMTP_SENDER_DOMAIN`     | Sender domain; must match the host portion of `SMTP_ADDR`.                                   |         |
| `SMTP_TIMEOUT`           | Send timeout.                                                                                | `20s`   |
| `SMTP_FORCE_UNENCRYPTED` | **Never set in production.** Allows plain credentials over an insecure connection; dev only. | `false` |

**REST tuning** (images `rest`, `standalone-rest`):

| Name                          | Description                          | Default          |
| ----------------------------- | ------------------------------------ | ---------------- |
| `REST_MAX_REQUEST_SIZE`       | Maximum request body size, in bytes. | `2097152` (2MiB) |
| `REST_TIMEOUT_READ`           | Read timeout.                        | `15s`            |
| `REST_TIMEOUT_READ_HEADER`    | Header read timeout.                 | `3s`             |
| `REST_TIMEOUT_WRITE`          | Write timeout.                       | `30s`            |
| `REST_TIMEOUT_IDLE`           | Idle keep-alive timeout.             | `60s`            |
| `REST_TIMEOUT_REQUEST`        | Per-request timeout.                 | `60s`            |
| `REST_TIMEOUT_SHUTDOWN`       | Graceful-shutdown timeout.           | `25s`            |
| `REST_CORS_ALLOWED_ORIGINS`   | CORS allowed origins.                | `*`              |
| `REST_CORS_ALLOWED_HEADERS`   | CORS allowed headers.                | `*`              |
| `REST_CORS_ALLOW_CREDENTIALS` | CORS allow-credentials flag.         | `false`          |
| `REST_CORS_MAX_AGE`           | CORS max-age, in seconds.            | `3600`           |

Database connection pool (server images). The limits are **per process**. The database's `max_connections` has to cover every replica plus the migration job; the stock `postgres` default is 100.

| Name                      | Description                               | Default |
| ------------------------- | ----------------------------------------- | ------- |
| `POSTGRES_MAX_OPEN_CONNS` | Maximum open connections to the database. | `20`    |
| `POSTGRES_MAX_IDLE_CONNS` | Maximum connections kept open while idle. | `20`    |

Logs and tracing — OpenTelemetry supports a stdout and a Google Cloud exporter (images `rest`, `jobs/init`, `standalone-rest`):

| Name                | Description                                                           | Default                  |
| ------------------- | --------------------------------------------------------------------- | ------------------------ |
| `OTEL`              | Enable OTel tracing; the variables below pick the exporter.           | `false`                  |
| `GCLOUD_PROJECT_ID` | Google Cloud project ID. When set, switches the OTel exporter to GCP. |                          |
| `APP_NAME`          | Application name attached to traces and logs.                         | `service-authentication` |

</details>

### Shutting down cleanly

On `SIGTERM` the server stops accepting, drains its open requests, then waits for the emails already
being sent. Registration and password-reset mail goes out on its own goroutine, so a send in progress
survives the request that triggered it and outlives the signal.

Three values have to increase in that order for the wait to complete:

```
SMTP_TIMEOUT (20s)  <  REST_TIMEOUT_SHUTDOWN (25s)  <  the deployment's termination grace period
```

Docker and Podman grant 10 seconds by default, which cuts the wait short and drops whatever is still
going out. `builds/podman-compose.yaml` raises it to 30s; a Kubernetes deployment needs
`terminationGracePeriodSeconds` set the same way.

## Using the client packages

Two clients ship with the service. Each snippet is the **minimum viable call**; the full surface is what your editor's intellisense, [pkg.go.dev](https://pkg.go.dev/github.com/a-novel/service-authentication/v2), and the [API reference](https://a-novel.github.io/service-authentication) are for.

- **Go** mounts an auth middleware — use it from a backend service that gates routes on roles and permissions. It needs a JSON Keys client to verify tokens, but no running authentication instance.
- **JavaScript / TypeScript** talks REST — use it from a frontend or Node service that drives the login / refresh / credential flows.

### Go

```bash
go get github.com/a-novel/service-authentication/v2
```

```go
package main

import (
	"context"
	"os"

	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	"github.com/go-chi/chi/v5"

	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
)

// Declare your roles. Each role grants a set of permissions; `inherits` pulls in
// another role's permissions transitively, and `priority` ranks roles for checks
// that compare two users (e.g. an admin acting on a lower-priority account).
var myPermissions = serviceauthentication.Permissions{
	// Keys must be roles your tokens actually carry. service-authentication issues the
	// built-in auth:* roles; a deployment can define more in its permissions config.
	Roles: map[string]serviceauthentication.Role{
		"auth:user":  {Priority: 0, Permissions: []string{"post:read"}},
		"auth:admin": {Priority: 1, Inherits: []string{"auth:user"}, Permissions: []string{"post:read", "post:write"}},
	},
}

func main() {
	ctx := context.Background()

	jsonKeysClient, _ := servicejsonkeys.NewClient("service-json-keys:8080")
	verifier := servicejsonkeys.NewClaimsVerifier[serviceauthentication.Claims](jsonKeysClient)
	logger := &loggingpresets.LogLocal{Out: os.Stdout}

	// withAuth gates routes on permissions.
	withAuth := serviceauthentication.NewAuthHandler(verifier, myPermissions, logger)
	router := chi.NewRouter()

	withAuth(router, "post:write").Get(...) // requires the post:write permission
	withAuth(router).Get(...)               // any authenticated (or anonymous) caller

	_ = ctx
}
```

### JavaScript / TypeScript

The package is published to GitHub Packages, which requires a Personal Access Token with the `read:packages` scope even for public packages ([why](https://github.com/orgs/community/discussions/23386#discussioncomment-3240193)). Add to `.npmrc` (project root or `$HOME`):

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

// Open an anonymous session — the entry point for register / login flows.
const token = await tokenCreateAnon(api);
```

Every method ships [zod](https://github.com/colinhacks/zod) request and response schemas, and responses are validated by default. API reference: [a-novel.github.io/service-authentication](https://a-novel.github.io/service-authentication).

## Running locally

For a throwaway instance without the dev toolchain, the **`standalone-rest`** image bundles the server, migrations, and the init bootstrap in one container. It runs migrations and init on every boot — handy for a quick spin-up, unsafe under multi-replica production restarts.

```yaml
services:
  postgres-authentication:
    image: ghcr.io/a-novel/service-authentication/database:v2.4.5
    networks: [api]
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256

  service-authentication:
    image: ghcr.io/a-novel/service-authentication/standalone-rest:v2.4.5
    ports: ["${SERVICE_AUTHENTICATION_REST_PORT}:8080"]
    depends_on:
      postgres-authentication: { condition: service_healthy }
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-authentication:5432/postgres?sslmode=disable"
      SERVICE_JSON_KEYS_HOST: "<json-keys-host>"
      SERVICE_JSON_KEYS_PORT: "<json-keys-grpc-port>"
    networks: [api]

networks:
  api:
```

Working on the service itself? Use the `a-novel` CLI (`a-novel run start service-authentication/rest`) instead — see [CONTRIBUTING](./CONTRIBUTING.md).

## Contributing

Platform setup and the day-to-day commands live in the [developer onboarding guide](https://github.com/a-novel-kit/.github/blob/master/README.md). Service-specific concepts and local interactions are in [CONTRIBUTING.md](./CONTRIBUTING.md).
