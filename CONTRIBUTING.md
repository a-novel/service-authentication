Welcome to the A-Novel project !

# Basics

- Learn Go: https://go.dev/tour/
- Learn PostgreSQL: https://www.postgresql.org/docs/current/tutorial.html
- Learn OpenAPI: https://learn.openapis.org/

Also take time to understand our frameworks:

- [Ogen](https://github.com/ogen-go/ogen): generate openAPI definitions for go
- [bun](https://bun.uptrace.dev/): PostgreSQL ORM for go.
- [JWT](https://a-novel-kit.github.io/jwt/): our go internal implementation of the RFC standard.
- [Mockery](https://github.com/vektra/mockery): mocking tool for go.

## Tools

- [Golang](https://go.dev/doc/install)
- [Node.js](https://nodejs.org/en/download/)
- [Python](https://www.python.org/downloads/)
  - Install [pipx](https://pipx.pypa.io/stable/installation/) to install command-line tools.
- [Podman](https://podman.io/docs/installation)

  - Install [podman-compose](https://github.com/containers/podman-compose)

    ```bash
    # Pipx
    pipx install podman-compose

    # Brew
    brew install podman-compose
    ```

- Make

  ```bash
  # Debian / Ubuntu
  sudo apt-get install build-essential

  # macOS
  brew install make
  ```

  For Windows, you can use [Make for Windows](https://gnuwin32.sourceforge.net/packages/make.htm)

# Scripts

## Lint

Make sure your codes always passes the required linters.

```shell
make format # Also fixes simple issues
make openapi-lint
```

## Generators

When you add new interfaces to `api` or `internal/services`, make sure they have proper mocks for testing:

```shell
make mocks
```

When updating OpenAPI spec, update ogen definitions:

```shell
make openapi-generate
```

# Project structure

The A-Novel authentication service is a REST API, built with OpenAPI and using a layered architecture (similar but not
exactly like the Clean Architecture).

> An arrow on the graph indicates the dependency flow. `A -> B` means that `A` may import / depend on `B`, but not the
> other way around. This is important to prevent circular dependencies.

```text
               ┌──────────────────────────────┐
               │ SCRIPTS                      │
               │                              ◄── External User
               │ Bash scripts for local tasks │
               └─────────────────┬────────────┘         │
                                 │                      │
                                 │               ┌──────▼──────┐
                                 │               │ CMD         │
                                 └───────────────►             │
                                                 │ Executables │
                                                 └──────┬──────┘
                                                        │
                    ┌───────────────────────────────────┼───────────────────────────┐
┌────────────────┐  │                          ┌────────▼─────────┐                 │
│ DOCS           │  │                          │ API              │                 │
│                ◄──┼──────────────────────────┤                  │                 │
│ Spec documents │  │                          │ OpenAPI handlers │                 │
└────────────────┘  │                          └──┬───────────────┘                 │
                    │                             │                                 │
                    │                             │                                 │
                    │  INTERNAL                   │                                 │
                    │                             │                                 │
                    │  Application Layers         │                                 │
                    │                             │                                 │
                    │ ┌───────────────────────────▼───┐      ┌────────────────────┐ │
                    │ │ SERVICES                      │      │ DAO                │ │
                    │ │                               ├──────►                    │ │
                    │ │ Business Logic implementation │      │ Data-Access Object │ │
                    │ └──────────────┬────────────────┘      └──────────┬─────────┘ │
                    │                │                                  │           │
                    │                │     ┌────────────────┐           │           │
                    │                │     │ LIB            │           │           │
                    │                └─────►                ◄───────────┘           │
                    │                      │ Internal tools │                       │
                    │                      └────────────────┘                       │
                    └──────┬─────────────────────┬────────────────────────┬─────────┘
                           │                     │                        │
     ┌─────────────────────▼──────┐ ┌────────────▼────────┐   ┌───────────▼────────┐
     │ MIGRATIONS                 │ │ CONFIG              │   │ MODELS             │
     │                            │ │                     ├───►                    │
     │ PostgreSQL migration files │ │ Configuration files │   │ Shared definitions │
     └────────────────────────────┘ └─────────────────────┘   └────────────────────┘
```

## Entry points

- `cmd`: Executables, they load dependencies and run a specific applications. Those are then loaded in the
  dockerfiles.
- `scripts`: Special shell scripts, used for local development.

## Setup

- `docs`: The OpenAPI spec
- `config`: Environmental dependencies, configured using `yaml` files.
- `models`: Shared definitions across the layers.
- `migrations`: PostgreSQL migration files.

## Implementation

- `api`: Handlers implemented from OpenAPI spec. Also exports ogen clients.
- `internal`: Unexported implementation.
  - `services`: Business logic.
  - `dao`: Data Access Object. The base class of this layer is called a Repository. A repository performs a single
    operation against a [bun model](https://bun.uptrace.dev/guide/models.html) called Entity.
  - `lib`: Internal tools, shared across layers.

# Contributing

## Discuss first

Discuss your contributions with the project maintainers for optimal guidance.

## Creating branches

Once you are ready to work, make sure you are up-to-date with the master branch

```shell
git checkout master
git pull
```

Then create a new branch with a descriptive name

```shell
git checkout -b type/my-feature
```

A branch name starts with a type:

- `feat`: add some new **user-facing** functionality
- `chores`: backend work that has no direct impact on the application (dependencies update, refactor, etc.)
- `security`: security-related fixes and work
- `fix`: bug fixes
- `community`: work related to the community (documentation, etc.)

The type is followed by a short description, that ideally matches the main commit message:

- Make it short (preferably, < 100 characters), while still being descriptive.
- kebab-case (lowercase, words separated by `-`)

## Opening Pull Requests

Once your work on a branch is ready, you may open a Pull Request. This will start the review process, and may be
followed by the merging of your work.
