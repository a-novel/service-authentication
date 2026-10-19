# Contributing to service-authentication

This guide is about working on the authentication service on your machine. Read the [README](./README.md) first for what the service does. The [developer guide](https://a-novel-kit.github.io/stack/) sets up the platform, and the [service concepts](https://github.com/a-novel/.github/blob/master/CONTRIBUTING.md) explain the layers every service shares.

---

## Quick local interactions

Start the REST server, then load the ports the `a-novel` daemon allocated into your shell:

```bash
a-novel run start service-authentication/rest
eval "$(a-novel run env service-authentication)"
AUTH_URL="http://localhost:${SERVICE_AUTHENTICATION_REST_PORT}"
MAIL_URL="http://localhost:${MAIL_UI_PORT}"
```

### Health

```bash
# Liveness
curl "$AUTH_URL/v2/ping"

# Dependency check (Postgres, JSON Keys, SMTP)
curl "$AUTH_URL/v2/healthcheck"
```

### Authentication flows

Most endpoints require a bearer token, and an anonymous session provides the first one. Set the account you will create first:

```bash
USER_EMAIL="you@example.com"
PASSWORD="choose-a-password"
```

```bash
ACCESS_TOKEN=$(curl -s -X PUT "$AUTH_URL/v2/session/anon" | jq -r '.accessToken')

# Inspect the current session.
curl "$AUTH_URL/v2/session" -H "Authorization: Bearer $ACCESS_TOKEN"
```

#### Register

Registration is invitation-only. The maintenance operation invites the address, and the invitation email carries a short code. [MailPit](#mailpit-email-testing) captures that email.

```bash
a-novel run exec service-authentication/maintenance -- \
  go run ./cmd/maintenance account-role --email "$USER_EMAIL" --role auth:user
```

Read the short code from the latest email, then create the account with it:

```bash
EMAIL_ID=$(curl -s "$MAIL_URL/api/v1/messages" | jq -r '.messages[0].ID')
SHORT_CODE=$(
  curl -s "$MAIL_URL/api/v1/message/$EMAIL_ID" |
    grep -o 'shortCode=[a-zA-Z0-9]*' | head -1 | cut -d= -f2
)

TOKEN=$(
  curl -s -X PUT "$AUTH_URL/v2/credentials" \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$USER_EMAIL\", \"password\": \"$PASSWORD\", \"shortCode\": \"$SHORT_CODE\"}"
)
ACCESS_TOKEN=$(jq -r '.accessToken' <<<"$TOKEN")
REFRESH_TOKEN=$(jq -r '.refreshToken' <<<"$TOKEN")
```

#### Login

```bash
TOKEN=$(
  curl -s -X PUT "$AUTH_URL/v2/session" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"$USER_EMAIL\", \"password\": \"$PASSWORD\"}"
)
ACCESS_TOKEN=$(jq -r '.accessToken' <<<"$TOKEN")
REFRESH_TOKEN=$(jq -r '.refreshToken' <<<"$TOKEN")
```

#### Refresh

```bash
TOKEN=$(
  curl -s -X PATCH "$AUTH_URL/v2/session" \
    -H "Content-Type: application/json" \
    -d "{\"accessToken\": \"$ACCESS_TOKEN\", \"refreshToken\": \"$REFRESH_TOKEN\"}"
)
ACCESS_TOKEN=$(jq -r '.accessToken' <<<"$TOKEN")
REFRESH_TOKEN=$(jq -r '.refreshToken' <<<"$TOKEN")
```

### MailPit (email testing)

[MailPit](https://mailpit.axllent.org/) captures every email the service sends locally, so none reaches a real inbox. Its UI listens on `$MAIL_URL`.

```bash
# Latest captured email
curl -s "$MAIL_URL/api/v1/messages" | jq '.messages[0]'

# Search by recipient
curl -s "$MAIL_URL/api/v1/search?query=to:$USER_EMAIL"

# Delete every captured email
curl -X DELETE "$MAIL_URL/api/v1/messages"
```

The [MailPit API reference](https://mailpit.axllent.org/docs/api-v1/view.html) lists the other endpoints.

---

## Service-specific concepts

### Two-token system

A session is a pair of JWTs. The [JSON Keys service](https://github.com/a-novel/service-json-keys) signs and verifies both, and this service holds no signing key.

| Token         | Purpose                             | Lifetime        |
| ------------- | ----------------------------------- | --------------- |
| Access token  | Authorizes API calls.               | Short (minutes) |
| Refresh token | Renews the pair without signing in. | Long (days)     |

A client signs in once and sends the access token until it expires. `PATCH /v2/session` then trades both tokens for a fresh pair. The access token names the refresh token that minted it, and only that refresh token renews it.

### Short codes

A short code is a single-use, time-limited secret, emailed to the user. Registration, an email change and a password reset each need one, so a session token alone never completes them. [`internal/config/auth/short_codes.config.yaml`](./internal/config/auth/short_codes.config.yaml) sets the code length and each usage's lifetime:

| Usage           | Flow                       | TTL   |
| --------------- | -------------------------- | ----- |
| `register`      | Account creation.          | `48h` |
| `validateEmail` | Email change confirmation. | `48h` |
| `resetPassword` | Password reset.            | `2h`  |

The database stores only a hash of each code. A consumed code is soft-deleted, which keeps it for the audit trail.

### Roles and permissions

[`internal/config/auth/permissions.config.yaml`](./internal/config/auth/permissions.config.yaml) defines the roles. Each role lists permissions and inherits those of the roles it names. Its priority ranks it when one user acts on another.

| Role              | Priority | Adds on top of inherited                                                         |
| ----------------- | -------- | -------------------------------------------------------------------------------- |
| `auth:anon`       | 0        | Complete a registration or an email change, reset a password, join the waitlist. |
| `auth:user`       | 1        | Change own password, request an email change.                                    |
| `auth:admin`      | 2        | Read, list and look up accounts.                                                 |
| `auth:superadmin` | 3        | Change roles, invite accounts.                                                   |

Routes check permissions, never roles. The Go middleware (`NewAuthHandler` in [`pkg/go`](./pkg/go)) resolves inheritance once at startup.

### Password security

Passwords are hashed with **Argon2id** (RFC 9106), in [`internal/lib/argon2.go`](./internal/lib/argon2.go). A plaintext password enters only through the registration, login and password-change request bodies, and is hashed before it reaches the database. Never log, return or store one anywhere else.

---

## Questions?

[Open an issue](https://github.com/a-novel/service-authentication/issues) with the logs and environment details.
