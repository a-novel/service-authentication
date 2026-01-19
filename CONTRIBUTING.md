# Contributing to service-authentication

Welcome to the authentication service for the A-Novel platform. This guide will help you understand the codebase, set
up your development environment, and contribute effectively.

## Table of Contents

1. [Introduction](#introduction)
2. [Quick Start](#quick-start)
3. [Project Architecture](#project-architecture)
4. [Development Workflow](#development-workflow)
5. [AI Usage](#ai-usage)
6. [Go Coding Conventions](#go-coding-conventions)
7. [OpenAPI Specification](#openapi-specification)
8. [Testing](#testing)
9. [CI/CD Pipeline](#cicd-pipeline)
10. [Project-Specific Guidelines](#project-specific-guidelines)
11. [Appendix](#appendix)

---

## Introduction

This is a **Go backend service** providing authentication for the A-Novel platform:

- User credential management (registration, authentication, password reset)
- JWT token generation and verification (two-token system: access + refresh)
- Role-based access control
- Email verification via short codes

The project also includes a **TypeScript client SDK** (`pkg/rest-js/`) for frontend integration, but the core service
is written entirely in Go.

**Tech Stack:**

| Layer          | Technology                                             |
| -------------- | ------------------------------------------------------ |
| Backend        | Go, Chi router, Bun ORM                                |
| Database       | PostgreSQL                                             |
| Authentication | JWT (via service-json-keys), Argon2id password hashing |
| Observability  | OpenTelemetry                                          |
| Containers     | Docker/Podman                                          |

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

## Project Architecture

### Directory Structure

```
service-authentication-v2/
├── cmd/                          # Entry points (Go)
│   ├── rest/main.go              # REST API server
│   ├── migrations/main.go        # Database migration runner
│   └── init/main.go              # Patch the server with initial state
│
├── internal/                     # Private Go packages (core logic)
│   ├── config/                   # Configuration management
│   ├── dao/                      # Data Access Objects (external data sources)
│   ├── handlers/                 # HTTP request handlers
│   │   └── middlewares/          # HTTP middleware
│   ├── services/                 # Business logic layer
│   ├── lib/                      # Utilities (cryptography, random)
│   └── models/                   # Domain models
│       ├── migrations/           # SQL migration files
│       └── mails/                # Email templates (MJML)
│
├── pkg/                          # Public packages
│   ├── auth.go                   # Public auth interface (Go)
│   ├── rest-js/                  # TypeScript REST client (wrapper)
│   └── rest-js-test/             # TypeScript test utilities
│
├── builds/                       # Docker build files
│   ├── *.Dockerfile              # Image definitions
│   └── podman-compose.yaml       # Local development stack
│
├── scripts/                      # Build and test scripts
└── .github/workflows/            # CI/CD pipelines
```

### Layered Architecture

The backend follows a clean layered architecture:

```
┌─────────────────────────────────────────────────────┐
│                     HTTP Layer                      │
│              (handlers/, middlewares/)              │
│  • Request parsing, validation, response formatting │
└──────────────────────────┬──────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────┐
│                    Service Layer                    │
│                     (services/)                     │
│  • Business logic, orchestration                    │
│  • Transaction management                           │
│  • Input validation, error handling                 │
└──────────────────────────┬──────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────┐
│                  Data Access Layer                  │
│                       (dao/)                        │
│  • External data source abstraction                 │
│  • Query execution and response mapping             │
└──────────────────────────┬──────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
   PostgreSQL            LLMs          Other sources
                    (AI prompting)    (caches, APIs...)
```

### Request Flow

```
HTTP Request
    │
    ▼
chi.Router (routing + middleware)
    │
    ▼
Handler.ServeHTTP()
    ├── Parse JSON body
    ├── Call service.Exec()
    └── Return JSON response
           │
           ▼
      Service.Exec()
           ├── Validate request
           ├── Execute business logic
           ├── Manage transactions (if any)
           └── Return result
                  │
                  ▼
             DAO.Exec()
                  ├── Query external data source
                  └── Map response to domain model
```

---

## Development Workflow

### Git Conventions

#### General Rules

- **Always lowercase** for branch names and commit messages
- `master` is the main branch — create feature branches from it
- One branch per feature/fix — keep changes focused

#### Branch Naming

Format: `type/scope/short-description`

```
feat/auth/two-factor-login
fix/dao/credentials-null-check
chore/deps/upgrade-chi
refactor/services/token-validation
```

**Types:**

| Type    | Usage                                       |
| ------- | ------------------------------------------- |
| `feat`  | New feature or capability                   |
| `fix`   | Bug fix                                     |
| `chore` | Maintenance (dependencies, config, tooling) |
| `test`  | Adding or updating tests                    |
| `perf`  | Performance improvements                    |

**Scopes** describe the affected area:

- **Layer**: `dao`, `services`, `handlers`, `cmd`, `config`
- **Domain**: `token`, `credentials`, `shortcode`, `auth`
- **General**: `deps`, `format`, `tests`, `ci`

#### Commit Messages

Format: `type(scope): short description`

```
feat(handlers): add rate limiting middleware
fix(dao): handle null email in credentials lookup
chore(deps): upgrade testify to v1.9.0
refactor(services): extract token validation logic
docs(readme): update installation instructions
test(credentials): add edge cases for email validation
perf(dao): optimize credentials list query
```

Keep descriptions concise and imperative ("add", "fix", "update" — not "added", "fixes", "updating").

#### Hotfixes

`hotfix` is **reserved for direct commits to `master`** in urgent situations. Do not use it for branch names.

```
hotfix(auth): patch token expiration vulnerability
```

Hotfixes bypass the normal PR flow and should be used sparingly for critical production issues only.

### Making Changes

1. **Create a branch** for your feature or fix
2. **Write tests first** (TDD encouraged)
3. **Implement the change**
4. **Run tests**: `make test`
5. **Run linters**: `make lint`
6. **Format code**: `make format`
7. **Create a pull request**

### Code Generation

The project uses code generation for:

- **Mocks**: Generated via `mockery` from interfaces
- **Email templates**: Compiled from MJML to HTML

Always run `make generate` after:

- Adding or modifying interfaces that need mocks
- Changing email templates in `internal/models/mails/`

### Database Migrations

Migrations live in `internal/models/migrations/` as embedded SQL files.

To add a new migration:

1. Create a new `.sql` file with a sequential name
2. Write your DDL statements
3. The migration runs automatically on service startup

---

## AI Usage

Using AI assistants (ChatGPT, Claude, Copilot, etc.) is allowed. They can significantly speed up development when used
appropriately. However, AI-generated code is still **your responsibility** — you own it, you maintain it, you debug it.

### Core Principles

1. **Use AI for code you understand and can review.** If you can't explain what the code does, don't commit it. AI is
   a tool to write code faster, not a replacement for understanding.

2. **Never trust AI blindly on critical logic.** Security-sensitive code (authentication, authorization, cryptography,
   input validation) requires thorough reading and understanding. AI can make subtle mistakes that create vulnerabilities.

3. **Verify before committing.** Always review AI-generated code for:
   - Correctness (does it actually do what you need?)
   - Consistency (does it follow project conventions?)
   - Security (does it introduce vulnerabilities?)
   - Completeness (did it miss edge cases?)

### When AI Works Well

- **Boilerplate and repetitive code**: CRUD operations, test cases, struct definitions
- **Syntax you don't remember**: Regex patterns, SQL queries, unfamiliar APIs
- **Documentation**: Comments, docstrings, README sections
- **Exploring approaches**: "How would I implement X?" as a starting point

### Practical Tips

- **Give context**: The more context you provide (existing code, conventions, requirements), the better the output
- **Iterate**: First drafts are rarely perfect — refine through follow-up prompts
- **Test immediately**: Run the code before assuming it works

---

## Go Coding Conventions

> This section contains patterns that apply to Go backend services in general. See [Project-Specific Guidelines](#project-specific-guidelines) for authentication-specific patterns.

### File Naming

| Type     | Pattern                   | Example                                        |
| -------- | ------------------------- | ---------------------------------------------- |
| Services | `{operation}.go`          | `tokenCreate.go`, `credentialsExist.go`        |
| Handlers | `http.{resource}.go`      | `http.credentials.go`, `http.token.go`         |
| DAO      | `{source}.{operation}.go` | `pg.credentialsList.go`, `openai.summarize.go` |
| Tests    | `{file}_test.go`          | `tokenCreate_test.go`                          |
| Mocks    | Generated in `mocks/`     | `mocks/mocks.go`                               |

### Function Naming

```go
// Constructors: New[Type]
func NewTokenCreate(deps Dependencies) *TokenCreate

// Main business method: Exec
func (s *TokenCreate) Exec(ctx context.Context, req *Request) (*Response, error)

// HTTP handlers: ServeHTTP
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request)

// Private helpers: camelCase
func (s *service) validateInput(req *Request) error
```

### Import Organization

Imports are organized in sections (enforced by `gci`):

```go
import (
    // 1. Standard library
    "context"
    "errors"
    "fmt"

    // 2. External packages
    "github.com/go-chi/chi/v5"
    "go.opentelemetry.io/otel"

    // 3. a-novel-kit packages
    "github.com/a-novel-kit/golib/otel"

    // 4. Project packages
    "github.com/a-novel/service-authentication/v2/internal/dao"

    // 5. Local (embed, etc.)
    _ "embed"
)
```

### Service Pattern

Every service follows this structure:

```go
// 1. Define interfaces for dependencies
type MyServiceRepository interface {
    GetData(ctx context.Context, id string) (*Data, error)
}

// 2. Define request/response structs
type MyServiceRequest struct {
    ID string `validate:"required,uuid"`
}

// 3. Define the service struct
type MyService struct {
    repository MyServiceRepository
}

// 4. Constructor
func NewMyService(repository MyServiceRepository) *MyService {
    return &MyService{repository: repository}
}

// 5. Main method with tracing and validation
func (s *MyService) Exec(ctx context.Context, request *MyServiceRequest) (*Response, error) {
    ctx, span := otel.Tracer().Start(ctx, "service.MyService")
    defer span.End()

    // Validate
    if err := validate.Struct(request); err != nil {
        return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
    }

    // Execute logic
    result, err := s.repository.GetData(ctx, request.ID)
    if err != nil {
        return nil, otel.ReportError(span, err)
    }

    return otel.ReportSuccess(span, result), nil
}
```

### Error Handling

```go
// Define package-level errors with Err prefix
var (
    ErrInvalidRequest = errors.New("invalid request")
    ErrNotFound       = errors.New("resource not found")
)

// Wrap errors with context
return nil, fmt.Errorf("fetch user: %w", err)

// Join errors when adding classification
return nil, errors.Join(err, ErrInvalidRequest)

// Check errors with errors.Is
if errors.Is(err, ErrNotFound) {
    // handle not found
}
```

### Handler Pattern

```go
type MyHandler struct {
    service MyServiceInterface
}

func NewMyHandler(service MyServiceInterface) *MyHandler {
    return &MyHandler{service: service}
}

func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    ctx, span := otel.Tracer().Start(r.Context(), "handler.MyHandler")
    defer span.End()

    // Parse request
    var request MyRequest
    if err := httpf.DecodeJSON(r, &request); err != nil {
        httpf.HandleError(ctx, w, span, nil, err)
        return
    }

    // Call service
    result, err := h.service.Exec(ctx, &request)
    if err != nil {
        httpf.HandleError(ctx, w, span, httpf.ErrMap{
            ErrNotFound: http.StatusNotFound,
            ErrInvalidRequest: http.StatusUnprocessableEntity,
        }, err)
        return
    }

    // Return response
    httpf.WriteJSON(ctx, w, span, result, http.StatusOK)
}
```

### Dependency Injection

All services receive dependencies via constructor:

```go
// Good: Dependencies injected
func NewService(repo Repository, logger Logger) *Service {
    return &Service{repo: repo, logger: logger}
}

// Bad: Global dependencies
func NewService() *Service {
    return &Service{repo: globalRepo}
}
```

### Validation

Validate at service layer entry, not in handlers:

```go
func (s *Service) Exec(ctx context.Context, req *Request) (*Response, error) {
    // Validate here
    if err := validate.Struct(req); err != nil {
        return nil, errors.Join(err, ErrInvalidRequest)
    }
    // Business logic...
}
```

### Database Conventions

#### Entity Naming

- Tables: `snake_case` plural (e.g., `credentials`, `short_codes`)
- Columns: `snake_case` (e.g., `created_at`, `email_validated`)

#### Bun ORM Tags

```go
type Credentials struct {
    bun.BaseModel `bun:"table:credentials"`

    ID             uuid.UUID  `bun:"id,pk,type:uuid,default:uuid_generate_v4()"`
    Email          string     `bun:"email,notnull,unique"`
    EmailValidated bool       `bun:"email_validated,notnull,default:false"`
    Password       string     `bun:"password,notnull"`
    Role           string     `bun:"role,notnull,default:'user'"`
    CreatedAt      time.Time  `bun:"created_at,notnull,default:current_timestamp"`
    UpdatedAt      *time.Time `bun:"updated_at"`
}
```

#### Embedded SQL (PostgreSQL DAO)

For PostgreSQL operations, DAO files use **raw SQL queries in external `.sql` files**, not Bun's query builder. This avoids typical ORM pitfalls (unexpected query generation, N+1 problems, abstraction leaks) and gives full control over the executed SQL:

```go
//go:embed pg.credentialsSelect.sql
var credentialsSelectQuery string

func (dao *CredentialsSelect) Exec(ctx context.Context, req *Request) (*Credentials, error) {
    var result Credentials
    err := dao.db.NewRaw(credentialsSelectQuery, req.ID).Scan(ctx, &result)
    return &result, err
}
```

Bun is used only for connection management and result scanning — the query itself is always explicit SQL.

### Email Templates

Templates use [MJML](https://mjml.io/) format in `internal/models/mails/`:

```
mails/
├── {templateName}.en.mjml    # English version
└── {templateName}.fr.mjml    # French version
```

After editing, regenerate HTML with:

```bash
pnpm generate
```

---

## OpenAPI Specification

The API is documented in `openapi.yaml` following the [OpenAPI 3.1 specification](https://spec.openapis.org/oas/v3.1.0). This is the source of truth for all available endpoints, request/response schemas, and security requirements.

### Manual Maintenance

The OpenAPI spec is **NOT auto-generated**. When adding or modifying endpoints, you must manually update `openapi.yaml` to reflect the changes:

- Add new paths and operations
- Update request/response schemas
- Document query parameters and headers
- Specify security requirements

### Editing

Use the [Swagger Editor](https://editor.swagger.io/) for visual editing with live validation and preview.

**Resources:**

- [OpenAPI 3.1 Specification](https://spec.openapis.org/oas/v3.1.0)
- [Swagger Editor](https://editor.swagger.io/)
- [OpenAPI Guide](https://swagger.io/docs/specification/about/)

### Validation

The spec is validated in CI using [Redocly](https://redocly.com/). Run locally:

```bash
pnpm lint:openapi
```

---

## Testing

### Running Tests

```bash
make test       # All tests
make test-unit  # Go unit tests only
```

### Table-Driven Tests

All tests use the **table-driven pattern**: test cases are defined as a slice of structs, each describing inputs, expected outputs, and mock behavior. This approach:

- **Centralizes test logic** — the assertion code is written once and reused for all cases
- **Makes adding cases trivial** — just append a new struct to the slice
- **Improves readability** — each case is self-documenting with a descriptive name
- **Enables parallel execution** — independent cases can run concurrently

```go
testCases := []struct {
    name      string
    request   *Request
    expect    *Response
    expectErr error
}{
    {name: "Success", request: &Request{ID: "123"}, expect: &Response{...}},
    {name: "Error/NotFound", request: &Request{ID: "bad"}, expectErr: ErrNotFound},
}

for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
        // test logic here
    })
}
```

### Test Case Naming

Use descriptive names that indicate the scenario:

| Pattern           | Usage                               |
| ----------------- | ----------------------------------- |
| `Success`         | Happy path                          |
| `Success/Variant` | Happy path with specific conditions |
| `Error/Reason`    | Expected failure with cause         |

Examples: `"Success"`, `"Success/WithOptionalField"`, `"Error/InvalidInput"`, `"Error/NotFound"`

### Parallel Execution

Use `t.Parallel()` to run tests concurrently, reducing total test time:

```go
func TestMyService(t *testing.T) {
    t.Parallel() // Run this test in parallel with other top-level tests

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel() // Run subtests in parallel with each other
            // ...
        })
    }
}
```

**When to use:** Services and handlers tests (with mocks) — these have no shared state.

**When to omit:** DAO tests wrapped in transactions — isolation is managed by the transaction, not parallelism.

### DAO Tests (Real Database)

DAO tests run against a real PostgreSQL instance to verify actual SQL behavior. Each test case is wrapped in a **transaction that rolls back automatically**, ensuring:

- Tests don't pollute each other's data
- No manual cleanup required
- Database state is reset between cases

```go
func TestCredentialsSelect(t *testing.T) {
    repository := dao.NewCredentialsSelect()

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
                // 1. Insert fixtures (test-specific data)
                // 2. Run the DAO method under test
                // 3. Assert results
                // Transaction auto-rollbacks after this function returns
            })
        })
    }
}
```

**Fixtures** are defined in the test case struct alongside expected inputs/outputs, then inserted at the start of the transactional callback:

```go
testCases := []struct {
    name     string
    fixtures []*dao.Credentials  // Data to insert before test
    request  *Request
    expect   *Response
}{
    {
        name:     "Success",
        fixtures: []*dao.Credentials{{ID: id1, Email: "user@test.com"}},
        request:  &Request{Email: "user@test.com"},
        expect:   &Response{ID: id1},
    },
    {
        name:     "Error/NotFound",
        fixtures: nil,  // Empty database
        request:  &Request{Email: "unknown@test.com"},
        expectErr: dao.ErrNotFound,
    },
}

// In the test loop:
postgres.RunTransactionalTest(t, config.PostgresPresetTest, func(ctx context.Context, t *testing.T) {
    // Insert fixtures using Bun's query builder directly
    for _, f := range tc.fixtures {
        _, err := db.NewInsert().Model(f).Exec(ctx)
        require.NoError(t, err)
    }

    // Then run the actual test
    result, err := repository.Exec(ctx, tc.request)
    // Assert...
})
```

Note: While production DAO code uses raw SQL (see [Embedded SQL](#embedded-sql-postgresql-dao)), fixtures use Bun's query builder directly. Inserting test data is a straightforward operation where ORM pitfalls don't apply, so we favor convenience here.

### Service/Handler Tests (Mocks)

Services and handlers are tested with **mocked dependencies**, making tests fast and deterministic. Mock behavior is defined per test case:

```go
testCases := []struct {
    name           string
    request        *Request
    repositoryMock *repositoryMockSetup // Define expected mock behavior
    expect         *Response
    expectErr      error
}{...}
```

Mocks are generated by mockery. After adding or modifying interfaces:

```bash
make generate
```

### Integration Tests (TypeScript)

Integration tests verify that the full service works correctly by making real HTTP requests against a running instance. These tests live in `pkg/test/rest-js/` and are written in TypeScript using Vitest.

#### Package Structure

The TypeScript packages are organized in three layers:

```
pkg/
├── rest-js/                 # Public SDK (@a-novel/service-authentication-rest)
│   └── src/                 # API client methods (tokenCreate, credentialsGet, etc.)
│
├── rest-js-test/            # Public test helpers (@a-novel/service-authentication-rest-test)
│   └── src/                 # Reusable test utilities (registerUser, checkEmail, etc.)
│
└── test/rest-js/            # Private test package (not published)
    └── src/                 # Actual integration test files (*.test.ts)
```

**Why this separation?**

- `rest-js` is the client SDK that frontend applications import to interact with the service
- `rest-js-test` contains helper functions that simplify common test scenarios (like registering a user, which involves multiple API calls and email verification). This package is **published** so other projects depending on this service can reuse these helpers in their own integration tests
- `pkg/test/rest-js` contains the actual test files. This package is **private** (not published) — it's only used to test this service

#### Writing Integration Tests

Integration tests use the SDK and test helpers together:

```typescript
import { AuthenticationApi, tokenCreate } from "@a-novel/service-authentication-rest";
import { preRegisterUser, registerUser } from "@a-novel/service-authentication-rest-test";

describe("credentialsCreate", () => {
  it("registers the user", async () => {
    const api = new AuthenticationApi(process.env.API_URL!);

    // Use test helpers for complex flows
    const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!, anonToken);
    await registerUser(api, preRegister);
  });
});
```

Test helpers handle multi-step operations (like email verification) so your tests stay focused on the behavior being tested.

#### Running Integration Tests

Integration tests require a running service instance:

```bash
make test-pkg-js
```

---

## CI/CD Pipeline

### Continuous Integration

The CI pipeline runs on every push and pull request:

```
┌─────────────────────────────────────────────────┐
│                 Generate Check                  │
│  • Verify go generate is up-to-date             │
│  • Verify pnpm generate is up-to-date           │
└─────────────────────┬───────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────┐
│                      Lint                       │
│  • golangci-lint (Go)                           │
│  • ESLint, Prettier, TypeScript (Node)          │
└─────────────────────┬───────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────┐
│                      Test                       │
│  • Go unit tests with coverage                  │
│  • Integration tests                            │
└─────────────────────┬───────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────┐
│                      Build                      │
│  • Docker images (database, migrations, etc.)   │
│  • TypeScript packages                          │
└─────────────────────┬───────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────┐
│                     Reports                     │
│  • Go Report Card                               │
│  • Codecov coverage                             │
│  • OpenAPI docs to GitHub Pages                 │
└─────────────────────────────────────────────────┘
```

### Deployment Strategy

We use **snapshot-based publishing** rather than true continuous delivery. Every commit triggers a build, but deployments are environment-specific:

| Source                    | Environment     | Description                                 |
| ------------------------- | --------------- | ------------------------------------------- |
| Feature branch commits    | **Feature env** | Isolated per-branch environment for testing |
| Untagged `master` commits | **Staging**     | Pre-production validation                   |
| Tagged versions           | **Production**  | Stable release snapshots                    |

This approach eliminates the need for a `develop` branch, preventing divergence issues between long-lived branches. All work flows directly from feature branches to `master`.

**Artifacts published on each commit:**

- Docker images (tagged with commit SHA or version)
- Go modules
- npm packages (to GitHub registry)

### Publishing a Release

To publish a new production release, use the publish scripts:

```bash
# Patch release (bug fixes): 2.2.1 → 2.2.2
pnpm publish:patch

# Minor release (new features): 2.2.1 → 2.3.0
pnpm publish:minor

# Major release (breaking changes): 2.2.1 → 3.0.0
pnpm publish:major
```

This will:

1. Bump version in all workspaces
2. Update version references in documentation
3. Commit and tag the release
4. Push to `master` with the new tag

The CI pipeline then builds and publishes all artifacts with the version tag.

### Docker Images

| Image        | Purpose                             |
| ------------ | ----------------------------------- |
| `database`   | PostgreSQL with extensions          |
| `migrations` | Schema migration runner             |
| `init`       | Patch the server with initial state |
| `rest`       | REST API server                     |
| `standalone` | All-in-one (dev only)               |

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

## Appendix

### A. Key Go Dependencies

| Package                                  | Purpose               |
| ---------------------------------------- | --------------------- |
| `github.com/go-chi/chi/v5`               | HTTP router           |
| `github.com/uptrace/bun`                 | PostgreSQL ORM        |
| `github.com/a-novel-kit/jwt`             | JWT handling          |
| `github.com/a-novel-kit/golib`           | Shared utilities      |
| `github.com/go-playground/validator/v10` | Struct validation     |
| `go.opentelemetry.io/otel`               | Observability         |
| `golang.org/x/crypto`                    | Cryptography (Argon2) |
| `github.com/stretchr/testify`            | Testing               |

### B. TypeScript Client SDK

The `pkg/rest-js/` directory contains a TypeScript client for frontend integration. This is a thin wrapper around the REST API and is rarely modified.

#### Conventions

- **Zod schemas** define request/response validation
- **Types are derived** from schemas using `z.infer<typeof Schema>`
- **One function per endpoint** with consistent signatures

#### When to Modify

Only modify the TypeScript client when:

- Adding a new API endpoint
- Changing an existing endpoint's request/response format
- Fixing a bug in the client

Run `make lint` and `make test` after changes.

---

## Questions?

If you have questions or run into issues:

- Open an issue at https://github.com/a-novel/service-authentication/issues
- Check existing issues for similar problems
- Include relevant logs and environment details
