# Contributing to service-authentication

For platform-wide setup (Go, Node, Podman, the `a-novel` CLI) and the day-to-day `a-novel` / `pnpm` commands, see the [developer onboarding guide](https://github.com/a-novel-kit/.github/blob/master/README.md). This file documents what is specific to the authentication service.

For deployment, configuration, and client-package integration, read the [README](./README.md) first. Contributors are expected to know what the service does and how operators run it before touching the code.

---

## Quick local interactions

Once the service is up (`a-novel run start service-authentication/rest`), the REST server listens on `${SERVICE_AUTHENTICATION_REST_PORT}` and MailPit's UI/API on `${MAIL_UI_PORT}`. Both ports are allocated by the `a-novel` daemon; inject them into your shell with `eval "$(a-novel run env service-authentication)"`.

### Health

```bash
# Liveness
curl http://localhost:${SERVICE_AUTHENTICATION_REST_PORT}/ping

# Dependency check (Postgres, JSON Keys, SMTP)
curl http://localhost:${SERVICE_AUTHENTICATION_REST_PORT}/healthcheck
```

### Authentication flows

Most endpoints require a bearer token. Start with an anonymous session, then register, log in, and refresh.

```bash
# For convenience, define the account you will create.
USER=<USER_EMAIL>
PASSWORD=<PASSWORD>
```

```bash
# Anonymous session — the entry point for register / login.
ACCESS_TOKEN=$(curl -X PUT http://localhost:${SERVICE_AUTHENTICATION_REST_PORT}/session/anon | jq -r '.accessToken')

# Inspect the current session.
curl -X GET http://localhost:${SERVICE_AUTHENTICATION_REST_PORT}/session \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

#### Register

Registration is a two-step, email-verified flow: request a short code, read it from the captured email, then create the account with it.

```bash
# Request a short code (emailed to the user).
curl -X PUT http://localhost:${SERVICE_AUTHENTICATION_REST_PORT}/short-code/register \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$USER\", \"lang\": \"en\"}"

# Pull the short code out of the latest captured email (see MailPit below).
EMAIL_ID=$(curl -s http://localhost:${MAIL_UI_PORT}/api/v1/messages | jq -r '.messages[0].ID')
SHORT_CODE=$(
  curl -s "http://localhost:${MAIL_UI_PORT}/api/v1/message/$EMAIL_ID" |
    grep -oP '(?<=shortCode\=)[a-zA-Z0-9]+' | head -1
)

# Complete registration with the code. Returns both tokens.
TOKEN=$(
  curl -X PUT http://localhost:${SERVICE_AUTHENTICATION_REST_PORT}/credentials \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$USER\", \"password\": \"$PASSWORD\", \"shortCode\": \"$SHORT_CODE\"}"
)
ACCESS_TOKEN=$(echo $TOKEN | jq -r '.accessToken')
REFRESH_TOKEN=$(echo $TOKEN | jq -r '.refreshToken')
```

#### Login

```bash
TOKEN=$(
  curl -X PUT http://localhost:${SERVICE_AUTHENTICATION_REST_PORT}/session \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$USER\", \"password\": \"$PASSWORD\"}"
)
ACCESS_TOKEN=$(echo $TOKEN | jq -r '.accessToken')
REFRESH_TOKEN=$(echo $TOKEN | jq -r '.refreshToken')
```

#### Refresh

```bash
TOKEN=$(
  curl -X PATCH http://localhost:${SERVICE_AUTHENTICATION_REST_PORT}/session \
    -H "Content-Type: application/json" \
    -d "{\"accessToken\": \"$ACCESS_TOKEN\", \"refreshToken\": \"$REFRESH_TOKEN\"}"
)
ACCESS_TOKEN=$(echo $TOKEN | jq -r '.accessToken')
REFRESH_TOKEN=$(echo $TOKEN | jq -r '.refreshToken')
```

### MailPit (email testing)

[MailPit](https://mailpit.axllent.org/) captures every email the service sends during local development — nothing reaches a real inbox. UI: http://localhost:${MAIL_UI_PORT}.

```bash
# Latest captured email (handy for grabbing a short code).
curl -s http://localhost:${MAIL_UI_PORT}/api/v1/messages | jq '.messages[0]'

# Clean slate.
curl -X DELETE http://localhost:${MAIL_UI_PORT}/api/v1/messages

# Search by recipient.
curl "http://localhost:${MAIL_UI_PORT}/api/v1/search?query=to:user@example.com"
```

See the [MailPit API v1 reference](https://mailpit.axllent.org/docs/api-v1/view.html) for more.

---

## Service-specific concepts

### Two-token system

Sessions are a pair of JWTs, both signed and verified through the [JSON Keys service](https://github.com/a-novel/service-json-keys) — this service holds no signing keys of its own.

| Token         | Purpose                         | Lifetime        |
| ------------- | ------------------------------- | --------------- |
| Access token  | Authorizes API calls.           | Short (minutes) |
| Refresh token | Mints a new pair without login. | Long (days)     |

A client authenticates once, then uses the access token until it expires and the refresh token to roll a fresh pair (`PATCH /session`) without re-sending credentials.

### Short codes

Single-use, time-limited codes that gate every identity-changing flow, emailed to the user so a session token alone can never complete them. Usages and TTLs live in [`internal/config/short_codes.config.yaml`](./internal/config/short_codes.config.yaml):

| Usage           | Flow                       | TTL   |
| --------------- | -------------------------- | ----- |
| `register`      | Account creation.          | `48h` |
| `validateEmail` | Email-change confirmation. | `48h` |
| `resetPassword` | Password reset.            | `2h`  |

A code is generated, emailed, consumed exactly once, then soft-deleted for the audit trail. The generated string length is the `size` field in the same file.

### Roles and permissions

Roles and their permissions are defined in [`internal/config/permissions.config.yaml`](./internal/config/permissions.config.yaml) and modelled by `config.Permissions` in [`internal/config/permissions.config.go`](./internal/config/permissions.config.go). Each role lists explicit permissions and may `inherit` another role's permissions transitively; `priority` ranks roles for checks that compare two users.

| Role              | Priority | Adds on top of inherited                              |
| ----------------- | -------- | ----------------------------------------------------- |
| `auth:anon`       | 0        | Register, request short codes, reset password.        |
| `auth:user`       | 1        | Patch own password, request email-update short codes. |
| `auth:admin`      | 2        | Read / list / check existence of credentials.         |
| `auth:superadmin` | 3        | Patch user roles.                                     |

Permissions are checked per route. The shipped Go middleware (`pkg/go.NewAuthHandler`, see the README) resolves inheritance at startup, so route mounts reference only leaf permissions.

### Password security

Passwords are hashed with **Argon2id** (RFC 9106). The implementation lives in [`internal/lib/argon2.go`](./internal/lib/argon2.go):

```go
// Hashing — pass the tuning parameters explicitly.
hash, err := lib.GenerateArgon2(password, lib.Argon2ParamsDefault)

// Verification — returns a non-nil error on mismatch.
err := lib.CompareArgon2(password, hash)
```

Plaintext passwords enter only through request bodies (registration, login, password change) and the `SUPER_ADMIN_PASSWORD` env var on the `init` job; each is hashed with Argon2 before it touches the database. Never log, expose, or persist them in plaintext anywhere else.

---

## Questions?

- Open an issue at https://github.com/a-novel/service-authentication/issues
- Check existing issues for similar problems
- Include relevant logs and environment details
