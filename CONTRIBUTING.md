# Contributing to service-authentication

This guide is about working on the authentication service on your machine. Read the [README](./README.md) first for what the service does. The [developer guide](https://a-novel-kit.github.io/stack/) sets up the platform, and the [taxonomy](https://github.com/a-novel/.github/blob/master/docs/taxonomy.md#inside-a-service-layers-and-contracts) explains the layers and contracts every service shares.

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

#### Sign out

Signing out takes a valid access token. When it has expired, run the refresh above first.

```bash
curl -s -X PUT "$AUTH_URL/v2/session/revoke" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"refreshToken\": \"$REFRESH_TOKEN\"}"
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

A signed-in client holds a pair of JWTs. The [JSON Keys service](https://github.com/a-novel/service-json-keys) signs and verifies both, and this service holds no signing key.

| Token         | Purpose                             | Lifetime   |
| ------------- | ----------------------------------- | ---------- |
| Access token  | Authorizes API calls.               | 15 minutes |
| Refresh token | Renews the pair without signing in. | 7 days     |

The JSON Keys configuration sets both lifetimes. A client signs in once and sends the access token until it expires. `PATCH /v2/session` then trades both tokens for a fresh pair. The access token names the refresh token that minted it, and only that refresh token renews it.

`PUT /v2/session/revoke` signs a session out. The caller's access token must belong to the same user as the refresh token. The route records the refresh token's ID in the `refresh_token_revocations` table, and a refresh then refuses that token. Access tokens the session already holds stay valid until they expire, since every service verifies them locally. A revocation row is useful until its refresh token expires, plus the verifier's clock-skew leeway. The scheduled `refresh-token-purge` maintenance operation then deletes it.

### Device sessions

A session is one sign-in on the account platform, by password or by account creation. The OAuth clients that sign the person in through it join it, so every token of the session carries one session ID: the JTI of the refresh token that opened it. The account platform shows each session as a device.

The `sessions` table gives each session a row, with the device the account platform describes in the `X-Client-User-Agent`, `X-Client-Region` and `X-Client-City` headers, beside the visitor IP that `REST_CLIENT_IP_HEADER` names. The row's last activity is when the session's latest access token was issued: at sign-in, at each `PATCH /v2/session`, and at each code or refresh grant of a client. A sign-out still lives in `refresh_token_revocations` alone. A session is active until it ends or a revocation holds its ID, and logged out afterwards.

| Route                                 | Effect                                                   |
| ------------------------------------- | -------------------------------------------------------- |
| `GET /v2/sessions`                    | Lists the caller's sessions, the active ones first.      |
| `PUT /v2/sessions/{sessionID}/revoke` | Signs one session out, the caller's own included.        |
| `PUT /v2/sessions/revoke-others`      | Signs out every other active session of the caller.      |
| `DELETE /v2/sessions/{sessionID}`     | Removes a logged-out session. An active one answers 409. |

Only the account platform manages devices, so an OAuth client's token gets 403 on these routes. A signed-out session keeps its access token until it expires, and these routes and `PATCH /v2/credentials/location-consent` refuse it at once with the 401 of `GET /v2/session`, so it cannot sign the account's other sessions out in turn. Each session's requests to each of these routes count against the windows of the login rate limits, per session, so no session of an account spends the budget of another.

The location consent is an account setting, off by default. While it is on, every session of the account records its IP address, region and city. `PATCH /v2/credentials/location-consent` sets it, and turning it off erases every stored location of the account at once.

A logged-out session stays listed until the account removes it. Each sign-in bounds the account's rows: it signs out the active sessions past the 100 active last, so the list shows every active session, then deletes the logged-out sessions past the 100 active last. The scheduled `refresh-token-purge` maintenance operation deletes the rows 30 days after their session ends, the `SessionRetention` of `internal/core`. A session ends at most the refresh token's 7 days after it starts, so a device signed out early stays listed up to 37 days. The purge reads the session's end, which never changes, so a renewal leaves the index it scans untouched.

### Single sign-on

A person signs in on the account platform alone. Every other platform is a client of it, and signs the person in with the OAuth 2.0 authorization code grant and PKCE. The browser never reaches this service.

1. The account platform checks the authorization request with `GET /v2/oauth/authorization-request`, then asks `POST /v2/oauth/authorization-code` for a code in the person's session.
2. The client trades the code and its PKCE verifier at the token endpoint, `POST /v2/oauth/token`, for a token pair in that session, and refreshes its access token there. Its refresh token renews for that client alone, and stops when the session ends. Signing out anywhere ends the session everywhere.

The OAuth routes follow RFC 6749, RFC 7636 and RFC 9700 to the letter. [`internal/core/oauth.go`](./internal/core/oauth.go) holds the parameter rules and the error codes, and each refusal is an `OAuthError` the handlers answer as an RFC error response. Keep it that way: a shortcut there, such as a JSON body or a prefix match on the redirect URI, is a security defect.

The `authorization_codes` table keeps the SHA-256 of each code alone. A code is valid one minute, and exchanges once. An exchanged row stays until it expires, so a second exchange finds it and ends the session. The scheduled `authorization-code-purge` maintenance operation then deletes it. A code signs an existing account in, and never creates one: accounts come from invitations alone.

The service tells the account platform and each client apart by the caller of its service token. A stack without `SERVICE_TOKEN_ISSUER`, which only `SERVICE_ISSUER_OPTIONAL` allows, verifies no caller: the two authorization routes answer 403 there, and the token endpoint `invalid_client`. The JavaScript integration stack runs the [`service-token-issuer` image](./README.md#service-tokens) and registers one client, `studio`. Its tests call as the account platform or as that client, so they sign a person in through the whole grant, as the platforms do.

A client's access token carries its `clientID`, and reaches the session routes and its own account alone. In [`cmd/rest/main.go`](./cmd/rest/main.go), `withAuth` refuses it with 403, and only `withSessionAuth` admits it, on `GET /v2/session`, `PUT /v2/session/revoke` and `GET /v2/credentials/self`, which lets a client name the person it signed in. Mount a new route with `withAuth`, so a client manages no account.

### Short codes

A short code is a single-use, time-limited secret, emailed to the user. Registration, an email change and a password reset each need one, so a session token alone never completes them. [`internal/config/auth/shortCodes.config.yaml`](./internal/config/auth/shortCodes.config.yaml) sets the code length and each usage's lifetime:

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
| `auth:user`       | 1        | Change own password, request an email change, sign in on a client.               |
| `auth:admin`      | 2        | Read, list and look up accounts.                                                 |
| `auth:superadmin` | 3        | Change roles, invite accounts.                                                   |

Routes check permissions, never roles. The Go middleware (`NewAuthHandler` in [`pkg/go`](./pkg/go)) resolves inheritance once at startup.

### Password security

Passwords are hashed with **Argon2id** (RFC 9106), in [`internal/lib/argon2.go`](./internal/lib/argon2.go). A plaintext password enters only through the registration, login and password-change request bodies, and is hashed before it reaches the database. Never log, return or store one anywhere else.

---

## Code license

The copyrightable source code that the Licensor owns in this repository is released under [RPL-1.5](LICENSE). [NOTICE.md](NOTICE.md) names the Licensor and holds the License Notice and the licensing history.

## Contributor agreement

Before submitting a contribution for inclusion in an official A-Novel repository, review the [Contributor License Agreement](CLA.md). The Project Owner may require a signed individual or corporate agreement before accepting the contribution. A pull request, a checkbox or a signed-off commit does not replace that signature.

If you developed the contribution for an employer or a client, or it includes third-party material, make sure you may grant the rights in the agreement, and disclose any third-party license terms.

The agreement gives the Project Owner additional rights over accepted contributions, including proprietary relicensing where lawful. It changes neither the RPL-1.5 license of public releases nor anything for the maintainers of independent forks who have not signed it.

## Submitting changes

1. Open an issue or pull request describing the proposed change.
2. Identify any incorporated third-party material and its license notices.
3. When asked, sign the agreement and identify your GitHub account, the repository and the pull request or commit it covers.
4. Keep existing copyright and license notices intact.

The Project Owner records signatures and matches them to accepted pull requests or commits. GitHub neither executes nor validates the agreement.

---

## Questions?

[Open an issue](https://github.com/a-novel/service-authentication/issues) with the logs and environment details.
