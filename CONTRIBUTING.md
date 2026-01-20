# Contributing to service-authentication

Welcome to the authentication service for the A-Novel platform. This guide will help you understand the codebase, set
up your development environment, and contribute effectively.

Before reading this guide, if you haven't already, please check the
[generic contribution guidelines](https://github.com/a-novel/.github/blob/master/CONTRIBUTING.md) that are relevant
to your scope.

---

## Quick Start

### Prerequisites

The following must be installed on your system.

- [Go](https://go.dev/doc/install)
- [Node.js](https://nodejs.org/en/download)
  - [pnpm](https://pnpm.io/installation)
- [Podman](https://podman.io/docs/installation)
- (optional) [Direnv](https://direnv.net/)
- Make
  - `sudo apt-get install build-essential` (apt)
  - `sudo pacman -S make` (arch)
  - `brew install make` (macOS)
  - [Make for Windows](https://gnuwin32.sourceforge.net/packages/make.htm)

### Bootstrap

Create a `.envrc` file in the project root:

```bash
cp .envrc.template .envrc
```

Ask for an admin to replace variables with a `[SECRET]` value.

Then, load the environment variables:

```bash
direnv allow .
# Alternatively, if you don't have direnv on your system
source .envrc
```

Finally, install the dependencies:

```bash
make install
```

### Common Commands

| Command         | Description                  |
| --------------- | ---------------------------- |
| `make run`      | Start all services locally   |
| `make test`     | Run all tests                |
| `make lint`     | Run all linters              |
| `make format`   | Format all code              |
| `make build`    | Build Docker images locally  |
| `make generate` | Generate mocks and templates |

### Interacting with the Service

Once the service is running (`make run`), you can interact with it using `curl` or any HTTP client.

#### Health Checks

```bash
# Simple ping (is the server up?)
curl http://localhost:4011/ping

# Detailed health check (checks database, dependencies)
curl http://localhost:4011/healthcheck
```

#### Authentication

Get an anonymous token (required for most interactions).

```bash
ACCESS_TOKEN=$(curl -X PUT http://localhost:4011/session/anon | jq -r '.accessToken')

# Verify session.
curl -X GET http://localhost:4011/session \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

Register an account.

```bash
# Create short code.
curl -X PUT http://localhost:4011/short-code/register \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -d '{"email": "newuser@example.com", "lang": "en"}'

# Retrieve email
EMAIL_ID=$(curl -s http://localhost:4014/api/v1/messages | jq -r '.messages[0].ID')
SHORT_CODE=$(
  curl -s "http://localhost:4014/api/v1/message/$EMAIL_ID" | \
    grep -oP '(?<=shortCode\=)[a-zA-Z0-9]+' | \
    head -1
)

# Complete registration with the code
TOKEN=$(
  curl -X PUT http://localhost:4011/credentials \
    -H "Authorization: Bearer $ACCESS_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"email\": \"newuser@example.com\", \"password\": \"securepassword\", \"shortCode\": \"$SHORT_CODE\"}"
)
ACCESS_TOKEN=$(echo $TOKEN | jq -r '.accessToken')
REFRESH_TOKEN=$(echo $TOKEN | jq -r '.refreshToken')
```

Refresh token on expiration.

```bash
# Refresh an expired access token
TOKEN=$(
  curl -X PATCH http://localhost:4011/session \
    -H "Content-Type: application/json" \
    -d "{\"accessToken\": \"$ACCESS_TOKEN\", \"refreshToken\": \"$REFRESH_TOKEN\"}"
)
ACCESS_TOKEN=$(echo $TOKEN | jq -r '.accessToken')
REFRESH_TOKEN=$(echo $TOKEN | jq -r '.refreshToken')
```

### MailPit (Email Testing)

[MailPit](https://mailpit.axllent.org/) captures all emails sent by the service during local development. No emails
are actually sent to real addresses.

**Access the UI:** http://localhost:4014

**Documentation:**

- [MailPit Features](https://mailpit.axllent.org/docs/)
- [API v1 Reference](https://mailpit.axllent.org/docs/api-v1/view.html)
- [Integration Testing Guide](https://mailpit.axllent.org/docs/integration/)

#### Quick API Examples

```bash
# List all captured emails
curl http://localhost:4014/api/v1/messages

# Get the latest email (useful for testing)
curl http://localhost:4014/api/v1/messages | jq '.messages[0]'

# Delete all emails (clean slate for testing)
curl -X DELETE http://localhost:4014/api/v1/messages

# Search for emails by recipient
curl "http://localhost:4014/api/v1/search?query=to:user@example.com"
```

---

## Project-Specific Guidelines

> This section contains patterns specific to this authentication service.

### Two-Token System

The service uses a two-token JWT system:

| Token         | Purpose           | Lifetime        |
| ------------- | ----------------- | --------------- |
| Access Token  | API authorization | Short (minutes) |
| Refresh Token | Session extension | Long (days)     |

**Flow:**

1. User authenticates with email/password
2. Service returns both tokens
3. Client uses access token for API calls
4. When access token expires, client uses refresh token to get new pair

### Short Codes

Temporary verification codes for:

- Email verification during registration
- Password reset
- Email change confirmation

**Lifecycle:**

1. Generated with expiration time
2. Sent to user via email
3. Consumed once (single-use)
4. Soft-deleted after use for audit trail

### Roles and Permissions

| Role         | Description               |
| ------------ | ------------------------- |
| `Anon`       | Anonymous/unauthenticated |
| `User`       | Authenticated user        |
| `Admin`      | Administrative access     |
| `SuperAdmin` | Full system access        |

Permissions are defined in `internal/config/permissions.go` and enforced via middleware.

### Password Security

Passwords are hashed using Argon2id (RFC 9106):

```go
// Hashing
hash, err := lib.GenerateArgon2(password)

// Verification
valid := lib.CompareArgon2(password, hash)
```

Never log, expose, or store plaintext passwords.

---

## Questions?

If you have questions or run into issues:

- Open an issue at https://github.com/a-novel/service-authentication/issues
- Check existing issues for similar problems
- Include relevant logs and environment details
