# Authentication service

[![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/agorastoryverse)](https://twitter.com/agorastoryverse)
[![Discord](https://img.shields.io/discord/1315240114691248138?logo=discord)](https://discord.gg/rp4Qr8cA)

<hr />

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/a-novel/service-authentication)
![GitHub repo file or directory count](https://img.shields.io/github/directory-file-count/a-novel/service-authentication)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/a-novel/service-authentication)

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/a-novel/service-authentication/main.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/a-novel/service-authentication)](https://goreportcard.com/report/github.com/a-novel/service-authentication)
[![codecov](https://codecov.io/gh/a-novel/service-authentication/graph/badge.svg?token=cnSwTJ2q4n)](https://codecov.io/gh/a-novel/service-authentication)

![Coverage graph](https://codecov.io/gh/a-novel/service-authentication/graphs/sunburst.svg?token=cnSwTJ2q4n)

## Usage

### Docker

Run the service as a containerized application (the below examples use docker-compose syntax).

```yaml
services:
  postgres-authentication:
    image: ghcr.io/a-novel/service-authentication/database:v2.1.9
    networks:
      - api
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - authentication-postgres-data:/var/lib/postgresql/

  service-authentication:
    image: ghcr.io/a-novel/service-authentication/standalone:v2.1.9
    ports:
      - "4011:8080"
    depends_on:
      postgres-authentication:
        condition: service_healthy
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-authentication:5432/postgres?sslmode=disable"
      SERVICE_JSON_KEYS_PORT: # Port where service-json-keys is running
      SERVICE_JSON_KEYS_HOST: # URL to a running service-json-keys instance
    networks:
      - api

networks:
  api:

volumes:
  authentication-postgres-data:
```

Note the standalone image is an all-in-one initializer for the application; however, it runs heavy operations such
as migrations on every launch. Thus, while it comes in handy for local development, it is NOT RECOMMENDED for
production deployments. Instead, consider using the separate, optimized images for that purpose.

```yaml
services:
  postgres-authentication:
    image: ghcr.io/a-novel/service-authentication/database:v2.1.9
    networks:
      - api
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: postgres
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - authentication-postgres-data:/var/lib/postgresql/

  migrations-authentication:
    image: ghcr.io/a-novel/service-authentication/migrations:v2.1.9
    depends_on:
      postgres-authentication:
        condition: service_healthy
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-authentication:5432/postgres?sslmode=disable"
    networks:
      - api

  # Optional job, used to inject base data into a freshly initialized database.
  init-authentication:
    image: ghcr.io/a-novel/service-authentication/init:v2.1.9
    depends_on:
      postgres-authentication:
        condition: service_healthy
      migrations-authentication:
        condition: service_completed_successfully
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-authentication:5432/postgres?sslmode=disable"
      # Create an initial super admin user. Make sure those credentials are passed in a secure manner.
      SUPER_ADMIN_EMAIL: # Email for the initial super admin user
      SUPER_ADMIN_PASSWORD: # Unencrypted password for the initial super admin user
    networks:
      - api

  service-authentication:
    image: ghcr.io/a-novel/service-authentication/rest:v2.1.9
    ports:
      - "4011:8080"
    depends_on:
      postgres-authentication:
        condition: service_healthy
      migrations-authentication:
        condition: service_completed_successfully
      init-authentication:
        condition: service_completed_successfully
    environment:
      POSTGRES_DSN: "postgres://postgres:postgres@postgres-authentication:5432/postgres?sslmode=disable"
      SERVICE_JSON_KEYS_PORT: # Port where service-json-keys is running
      SERVICE_JSON_KEYS_HOST: # URL to a running service-json-keys instance
    networks:
      - api

networks:
  api:

volumes:
  authentication-postgres-data:
```

Above are the minimal required configuration to run the service locally. Configuration is done through environment
variables. Below is a list of available configurations:

**Required variables**

| Name                   | Description                                                          | Images                                              |
| ---------------------- | -------------------------------------------------------------------- | --------------------------------------------------- |
| POSTGRES_DSN           | The Postgres Data Source Name (DSN) used to connect to the database. | `standalone`<br/>`rest`<br/>`init`<br/>`migrations` |
| SERVICE_JSON_KEYS_PORT | Port where service-json-keys is running                              | `standalone`<br/>`rest`                             |
| SERVICE_JSON_KEYS_HOST | URL to a running service-json-keys instance                          | `standalone`<br/>`rest`                             |

This service requires a running instance of the [JSON Keys service](https://github.com/a-novel/service-json-keys). Note
that the authentication and json keys service share sensitive data, they should communicate over a secure, unexposed
network.

**Platform connection**

You can provide a connection to the client platform by pointing to a running instance of the
[authentication platform](https://github.com/a-novel/platform-authentication).

This connection is optional and only used to populate links in emails.

| Name                              | Description                                                            | Default value                             | Images                  |
| --------------------------------- | ---------------------------------------------------------------------- | ----------------------------------------- | ----------------------- |
| PLATFORM_AUTH_URL                 | URL to the client platform (optional if all other values are provided) |                                           | `standalone`<br/>`rest` |
| PLATFORM_AUTH_URL_UPDATE_EMAIL    | URL to the client platform email validation page                       | PLATFORM_AUTH_URL + `/ext/email/validate` | `standalone`<br/>`rest` |
| PLATFORM_AUTH_URL_UPDATE_PASSWORD | URL to the client platform password update page                        | PLATFORM_AUTH_URL + `/ext/password/reset` | `standalone`<br/>`rest` |
| PLATFORM_AUTH_URL_REGISTER        | URL to the client platform register page                               | PLATFORM_AUTH_URL + `/ext/account/create` | `standalone`<br/>`rest` |

**SMTP**

By default, this service uses a mock email sender that prints email content to the standard
output. It is highly recommended to set an actual SMTP server in production environments,
as emails contain sensitive information (eg short codes).

| Name                   | Description                                                                                                                                                                                                                 | Default value | Images                  |
| ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------- | ----------------------- |
| SMTP_ADDR              | Address of the SMTP server (`domain:port`)                                                                                                                                                                                  |               | `standalone`<br/>`rest` |
| SMTP_SENDER_NAME       | Name that will appear as the sender in outgoing emails                                                                                                                                                                      |               | `standalone`<br/>`rest` |
| SMTP_SENDER_EMAIL      | Email address used to send outgoing smtp emails                                                                                                                                                                             |               | `standalone`<br/>`rest` |
| SMTP_SENDER_PASSWORD   | Plain password used to connect to the sender email account, this data is sensitive so handle with care                                                                                                                      |               | `standalone`<br/>`rest` |
| SMTP_SENDER_DOMAIN     | Domain used for sending Smtp emails                                                                                                                                                                                         |               | `standalone`<br/>`rest` |
| SMTP_TIMEOUT           | Set the timeout for sending emails                                                                                                                                                                                          | `20s`         | `standalone`<br/>`rest` |
| SMTP_FORCE_UNENCRYPTED | DO NOT SET IN PRODUCTION. This variable bypasses SMTP security by allowing plain credentials over insecure connections. This setting is intended for development only, and could compromise your credentials in production. | `false`       | `standalone`<br/>`rest` |

**Rest API**

While you should not need to change these values in most cases, the following variables allow you to
customize the API behavior.

| Name                       | Description                                 | Default value    | Images                  |
| -------------------------- | ------------------------------------------- | ---------------- | ----------------------- |
| API_MAX_REQUEST_SIZE       | Maximum size of incoming requests in bytes  | `2097152` (2MiB) | `standalone`<br/>`rest` |
| API_TIMEOUT_READ           | Timeout for read operations                 | `15s`            | `standalone`<br/>`rest` |
| API_TIMEOUT_READ_HEADER    | Timeout for header reading operations       | `3s`             | `standalone`<br/>`rest` |
| API_TIMEOUT_WRITE          | Timeout for write operations                | `30s`            | `standalone`<br/>`rest` |
| API_TIMEOUT_IDLE           | Idle timeout                                | `60s`            | `standalone`<br/>`rest` |
| API_TIMEOUT_REQUEST        | Timeout for api requests                    | `60s`            | `standalone`<br/>`rest` |
| API_CORS_ALLOWED_ORIGINS   | CORS allowed origins (allow all by default) | `*`              | `standalone`<br/>`rest` |
| API_CORS_ALLOWED_HEADERS   | CORS allowed headers (allow all by default) | `*`              | `standalone`<br/>`rest` |
| API_CORS_ALLOW_CREDENTIALS | CORS allow credentials                      | `false`          | `standalone`<br/>`rest` |
| API_CORS_MAX_AGE           | CORS max age                                | `3600`           | `standalone`<br/>`rest` |

**Logs & Tracing**

For now, OTEL is only provided using 2 exporters: stdout and Google Cloud. Other integrations may come
in the future.

| Name              | Description                                                                             | Default value            | Images                             |
| ----------------- | --------------------------------------------------------------------------------------- | ------------------------ | ---------------------------------- |
| OTEL              | Activate OTEL tracing (use options below to switch between exporters)                   | `false`                  | `standalone`<br/>`rest`<br/>`init` |
| GCLOUD_PROJECT_ID | Google Cloud project id for the OTEL exporter. Switch to Google Cloud exporter when set |                          | `standalone`<br/>`rest`<br/>`init` |
| APP_NAME          | Application name to be used in traces                                                   | `service-authentication` | `standalone`<br/>`rest`<br/>`init` |

**Setup**

The below variables allow you to setup an empty instance through a job, injecting the minimum data to
use the application.

| Name                 | Description                                           | Images                  |
| -------------------- | ----------------------------------------------------- | ----------------------- |
| SUPER_ADMIN_EMAIL    | Email for the initial super admin user                | `standalone`<br/>`init` |
| SUPER_ADMIN_PASSWORD | Unencrypted password for the initial super admin user | `standalone`<br/>`init` |

### Javascript (npm)

To interact with a running instance of the authentication service, you can use the integrated package.

> ⚠️ **Warning**: Even though the package is public, GitHub registry requires you to have a Personal Access Token
> with `repo` and `read:packages` scopes to pull it in your project. See
> [this issue](https://github.com/orgs/community/discussions/23386#discussioncomment-3240193) for more information.

Make sure you have a `.npmrc` with the following content (in your project or in your home directory):

```ini
@a-novel:registry=https://npm.pkg.github.com
@a-novel-kit:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${YOUR_PERSONAL_ACCESS_TOKEN}
```

Then, install the package using pnpm:

```bash
# pnpm config set auto-install-peers true
#  Or
# pnpm config set auto-install-peers true --location project
pnpm add @a-novel/service-authentication-rest
```

To use it, you must create an `AuthenticationApi` instance. A single instance can be shared across
your client.

```typescript
import { AuthenticationApi, tokenCreateAnon } from "@a-novel/service-authentication-rest";

export const authenticationApi = new AuthenticationApi("<base_api_url>");

// (optional) check the status of the api connection.
await authenticationApi.ping();
await authenticationApi.health();
```

You can then call methods from the package using this api instance. Each method comes with
[zod](https://github.com/colinhacks/zod) types so you can validate requests easily.

Responses are validated by default.

```typescript
import {
  ClaimsSchema,
  CredentialsCreateRequestSchema,
  CredentialsExistsRequestSchema,
  CredentialsGetRequestSchema,
  CredentialsListRequestSchema,
  CredentialsResetPasswordRequestSchema,
  CredentialsSchema,
  CredentialsUpdateEmailRequestSchema,
  CredentialsUpdatePasswordRequestSchema,
  CredentialsUpdateRoleRequestSchema,
  ShortCodeCreateEmailUpdateRequestSchema,
  ShortCodeCreatePasswordResetRequestSchema,
  ShortCodeCreateRegisterRequestSchema,
  TokenCreateRequestSchema,
  TokenRefreshRequestSchema,
  TokenSchema,
  claimsGet,
  credentialsCreate,
  credentialsExists,
  credentialsGet,
  credentialsList,
  credentialsResetPassword,
  credentialsUpdateEmail,
  credentialsUpdatePassword,
  credentialsUpdateRole,
  shortCodeCreateEmailUpdate,
  shortCodeCreatePasswordReset,
  shortCodeCreateRegister,
  tokenCreate,
  tokenCreateAnon,
  tokenRefresh,
} from "@a-novel/service-authentication-rest";
```

### Go module

You can integrate the authentication capabilities directly into your Go services by using the provided
Go module. Note this does not require a running instance of the authentication service, but only of its
[JSON Keys service](https://github.com/a-novel/service-json-keys) dependency.

```bash
go get -u github.com/a-novel/service-authentication/v2
```

```go
package main

import (
	"context"

	"github.com/go-chi/chi/v5"

	authpkg "github.com/a-novel/service-authentication/v2/pkg"
	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"
)

// Define roles for your application (required).
//
// The key is the name of the role, as it is passed to the JWT payload.
// Permissions are evaluated at path level, which means a given role can
// only access resources for which it has explicit permissions. For more
// fine-grained access, you must implement custom validation yourself.
//
// The priority argument serves as a hierarchy indicator between roles, and is
// used by some custom access checks to grant permission for an operation between
// 2 users, based on their relative roles priorities.
var myPermissions = authpkg.Permissions{
	Roles: map[string]authpkg.Role{
		"role1": {
			Priority: 0,
			Permissions: []string{"permission1", "permission2"},
        },
		"role2": {
			Priority: 1,
			Inherits: []string{"role1"},
			Permissions: []string{"permission3"},
		},
		"role3": {
			Priority: 0,
			Permissions: []string{"permission4"},
		},
    },
}

func main() {
	ctx := context.Background()

	jsonKeysClient, _ := jkpkg.NewClient("<service-json-keys-url>")
	serviceVerifyAccessToken := jkpkg.NewClaimsVerifier[authpkg.Claims](jsonKeysClient)

	// You can now add permission-based authentication to your routes.
	withAuth := authpkg.NewAuthHandler(serviceVerifyAccessToken, myPermissions)
	router := chi.NewRouter()

	// Route only accessible to users with role2.
	withAuth(router, "permission3").Get(...)
	// Route accessible to users with role1 or role2.
	withAuth(router, "permission2").Get(...)
	// Route accessible to all authenticated users.
	withAuth(router).Get(...)
	// Route accessible to users with role2 or role3.
	withAuth(router, "permission3", "permission4").Get(...)
}
```
