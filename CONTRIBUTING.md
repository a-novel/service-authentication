Welcome to the A-novel authentication service!

- [Prepare local environment](#prepare-local-environment)
- [Verify the code](#verify-the-code)
- [project Architecture](#project-architecture)
- [Testing](#testing)

# Prepare local environment

Install the required development tools:

- [Golang](https://go.dev/doc/install)
- [Podman](https://podman.io/docs/installation)
- [npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm): used to lint the OpenAPI spec.
- Make
  ```bash
  # Debian / Ubuntu
  sudo apt-get install build-essential
  
  # macOS
  brew install make
  ```
  For Windows, you can use [Make for Windows](https://gnuwin32.sourceforge.net/packages/make.htm)

Run the API using instructions from the main README.md file.

```bash
make rotate_keys
# 9:07PM INF key generated app=authentication job=rotate-keys key_id=... usage=auth
# 9:07PM INF no key generated app=authentication job=rotate-keys usage=refresh
# 9:07PM INF rotation done app=authentication failed_keys=0 generated_keys=1 job=rotate-keys total_keys=2
```

```bash
make api
# 3:09PM INF starting application... app=authentication
# 3:09PM INF application started! address=:4001 app=authentication
```

# Verify the code

Those steps are important requirements that every new piece of code should pass. The following command MUST succeed
each time you want to commit a change:

```bash
# This command also fixes some lint issues automatically,
# so run it as often as you can.
make format
```

```bash
make test
```

If you modify the OpenAPI spec, you must run an extra check:

```bash
make openapi-lint
# Woohoo! Your API description is valid. 🎉
```

# project Architecture

The A-Novel authentication service is a REST API, built with OpenAPI and using a layered architecture (similar but not
exactly like the Clean Architecture).

> An arrow on the graph indicates the dependency flow. `A -> B` means that  `A` may import / depend on `B`, but not the
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
               │  ┌────────────────┐      ┌────────▼─────────┐                 │
               │  │ DOCS           │      │ API              │                 │
               │  │                ◄──────┤                  │                 │
               │  │ Spec documents │      │ OpenAPI handlers │                 │
               │  └────────────────┘      └──┬───────────────┘                 │
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

The core layers you'll likely get involved with are:

- [DAO](#data-access-object-dao): Data-Access Object, the layer that interacts with the database.
- [Services](#services): Business logic layer.
- [API](#api): The main I/O of the application.
- [Models](#models): Shared types and definitions.
- [CMD](#cmd): Executables that can be run from the command line. You'll likely make adjustements in those to reflec
  your modifications on the other layers.

You should have fewer interactions with the following layers:

- [Config](#config): Static definitions that can be configured without touching the business code.
- [Lib](#lib): Internal tools that can be reused across the application.
- [Scripts](#scripts): Local tasks that can be executed from the command line.
- [Migrations](#migrations): PostgreSQL migration files.

## Data-Access Object (DAO)

The DAO layer is an abstraction between the [models](#models), and a data source (dbms, file system, etc.).

The interfaces used to interact with the data sources are called Repositories. Each repository describes a single
action on a single data model, called Entity.

<hr/>

Considerations:

We use [bun](https://bun.uptrace.dev/) as an ORM for PostgreSQL. While there are many arguments against the use of
ORMs, we focus on simple queries over advanced usage that would require more tuning.

Because we mainly do CRUD, the performance gains from using raw SQL are negligible. We follow a few rules to keep
things simple, and avoid the necessity of pure SQL:

- **No Dependencies**: we avoid foreign keys and such as much as possible, so `JOIN` queries are almost never needed.
  We prefer data duplication when information is needed at vastly different places.
- **Abstract with views**: bun allows to define separate select tables on a model, which can be abstracted views over
  the actual source. This allows to displace complex SQL filtering directly in the migration, and keep using bun with  
  simple CRUD queries.

<hr/>

```go
package dao

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"
)

type DeleteFooData struct {
}

type DeleteFooRepository struct{}

func (repository *DeleteFooRepository) DeleteFoo(ctx context.Context, data DeleteFooData) (*FooEntity, error) {
	tx, err := pgctx.Context(ctx)
	if err != nil {
		return nil, fmt.Errorf("(DeleteFooRepository.DeleteFoo) get postgres client: %w", err)
	}

	entity := &FooEntity{}

	// Execute query.
	res, err := tx.NewUpdate().Model(entity).(...).Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("(DeleteFooRepository.DeleteFoo) delete foo: %w", err)
	}

	return entity, nil
}

func NewDeleteFooRepository() *DeleteFooRepository {
	return &DeleteFooRepository{}
}
```

Entities are go struct, declared with the fields and tags specific to their data source. For example, Postgres entities
might declare bun-related fields.

```go
package dao

import (
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type FooEntity struct {
	bun.BaseModel `bun:"table:foos,select:active_foos"`

	ID uuid.UUID `bun:"id,pk,type:uuid"`

	//...
}
```

A repository ALWAYS takes a `context.Context` and a source data as arguments. The source data is usually a struct
specific to the repository, that shares its naming pattern. Extra arguments may be defined, and required along the
source data. If so, those extra arguments must be passed between the `context.Context` and the source data.

```go
func (repository *ActionFooRepository) ActionFoo(ctx context.Context, ...extraArgs, data ActionData) (*FooEntity, error)
```

A repository ALWAYS return a result and an error. The result can be anything, but is usually an entity, or a list of
entities.

## Services

Services are interfaces that contain the business logic of the application. They are the spine of every other layer.

```go
package services

import "github.com/a-novel-kit/context"

type FooServiceSource interface {
}

type FooServiceRequest struct {
}

type FooService struct {
	source FooServiceSource
}

func (service *FooService) Foo(ctx context.Context, request FooServiceRequest) (*models.SomeModel, error) {

}

func NewFooService(source FooServiceSource) *FooService {
	return &FooService{
		source: source,
	}
}
```

A service is built over a source. A source is an abstraction of the interfaces exported by other layers, that are
required to perform the service's actions. It can also abstract other services, since a service may depend on one or
more other services.

> Unless a source abstract a single interface (for example a DAO), it may be convenient to provide a constructor for
> the source. This constructor is used for generating a new actual implementation (non-mocked) of the source.
>
> ```go
> package services
>
> type FooServiceSource interface {
> 	Method1()
> 	Method2()
> }
>
> // Type1 implements Method1
> // Type2 implements Method2
> func NewFooServiceSource(
> 	type1 Type1,
> 	type2 Type2,
> ) FooServiceSource {
> 	// Use the anonymous fields property, that allows a struct to
> 	// inherit the methods of its unnamed fields.
> 	return &struct {
> 		Type1
> 		Type2
> 	}{
> 		Type1: type1,
> 		Type2: type2,
> 	}
> }
> ```

The service then process a request using the source, and returns a result. The returned result can either be a model,
or a locally defined type.

## API

The main I/O of the application. The API uses [OpenAPI](https://github.com/OAI/OpenAPI-Specification) definitions
written in the `docs/api.yaml` file.

Those definitions are then parsed using [ogen](https://github.com/ogen-go/ogen), to generate a Go SDK.

When a definition is updated, the `api.Handler` interface will be as well. Each method is implemented in a separate
file to keep code clean, using the `api.API` struct we defined.

```go
package api

import (
	"errors"
	"fmt"
	"github.com/a-novel-kit/context"
)

func (api *API) Action(ctx context.Context, params ActionParams) (ActionRes, error) {
	res, err := api.ActionService.Action(ctx, ...)

	switch {
	case errors.Is(err, SomeError):
		return &SomeExpectedError{}, nil
	case err != nil:
		return nil, fmt.Errorf("action: %w", err)
	}

	return &ActionResOK{...}, nil
}
```

> Adding a new action usually includes adding a new service on the API struct as well. Then, this service must be
> passed on the API from the `cmd` layer.

## Models

Shared types and definitions are declared in the `models` package. This package only contains static definitions, and
dynamic / functional logic should be avoided as much as possible.

## CMD

This layer contains go files that can be run as executables. The struct of the `cmd` directory is as follow:

```text
cmd
├── command_1
│   └── main.go
└── command_2
    └── main.go
```

Each `main.go` file contains a `main()` func that initializes the dependencies, and runs (either services for a job
or an API).

## Config

The config directory contains static definitions, that usually depend on the environment and can be configured without
touching the business code.

The `config` package may contain any of the following:

- **Initializers**: some modules require to initialize a global, shared variable. This variable, if it only depends
  on static information, can be initialized in the `config` package.
- **Configuration Objects**: A configuration object loads static values from the environment. Those objects are loaded
  from `.yaml` files, using the [configuration module](https://a-novel-kit.github.io/configurator/).

> IMPORTANT: configuration object files MUST not contain hardcoded secrets. Secret values, such as API keys, must be
> loaded as environment variables. In this case, the value in the `.yaml` file should look like this:
> ```yaml
> api_key: ${API_KEY}
> ```

> Passing a secret value also requires to update the deployment workflow, so the value can be passed at build time from
> the cloud provider.

## Lib

Lib contains internal tools that can be reused across the application. Their purpose is not specific to a layer,
although is must match a use-case that is at least partially scoped to the current service. Otherwise, it should be
extracted to a separate, shared package.

There aren't rules as to how a lib element should be written, nor what it should contain. The only constraint is that
it must respond to a generic use case, independent of a specific layer.

## Scripts

Every local action is executed from a `Makefile`, located at the project root.

<hr/>

Considerations:

When writing an action in the Makefile, try to avoid unnecessary tools installations. Many package manager allow you
to run executable and install them automatically if needed, removing this preoccupation from the user.

```shell
# ❌ Don't
npm install -g tool
tool ...

# ✅ Do
npx tool ...
```

```shell
# ❌ Don't
go install github.com/org/module/tool@version
tool ...

# ✅ Do
go run github.com/org/module/tool@version ...
```

> If you need to add a dependency, such as a package manager, update the readme's accordingly. They usually list
> external tools that are required to run the project.

<hr/>

The makefile can sometimes make use of a bash script located under the `scripts` directory.

Each action in the makefile must have a comment describing it.

```makefile
# My action does this
action:
    echo "Hello world!"
```

If your action needs to have the name of an existing directory, you must declare it in the `.PHONY` section of the
makefile.

```makefile
action:
    echo "Hello world!"
    
.PHONY: action
```

## Migrations

A-Novel uses the [PostgreSQL](https://www.postgresql.org/) database. This is a relational database that requires
defined schemas. When you need to modify or extend those schemas, you must create a new pair of migration files.

### Migrations naming convention

```
[timestr]_[action]
```

The `timestr` is a timestamp in the format `YYYYMMDDHHMMSS`. It is used to order the migrations.

You can use this command to retrieve the current `timestr` in the correct format:

```shell
date '+%Y%m%d%H%M%S'
```

The action describes what you are adding / changing with this migration, in snake_case.

> Scope migrations to only a single actions, and prefer multiple separate migrations over a complex one.

### Up and Down migrations

Each migration must come with an UP and DOWN version. Files are respectively named `[name].up.sql` and
`[name].down.sql`.

The UP migration applies your changes, while the DOWN migration reverts them.

```sql
/* UP migration */
CREATE
SOMETHING;

/* DOWN migration */
DROP
SOMETHING;
```

Be careful of order execution: usually, you must first revert changes that were applied last, to avoid conflicting
states:

```sql
/* UP migration */
CREATE
SOMETHING_1;
CREATE
SOMETHING_2;

/* DOWN migration */
DROP
SOMETHING_2;
DROP
SOMETHING_1;
```

### Apply migrations

Migrations are automatically applied when any command is executed.

# Testing

The `dao`, `services` and `lib` layers are FULLy unit-tested. Static layers like `models` and `config` are usually
ignored. The `cmd` and `api` may be covered by integration tests.

To ensure the isolation of tests, test file MUST declare a separate package with the `_test` suffix.

For example, if the test file is in the `layer` package:

```go
package layer_test
```

## Table-driven pattern

Whenever possible, tests use the [Table-Driven pattern](https://go.dev/wiki/TableDrivenTests) recommended by the Go
documentation.

```go
package layer_test

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSomething(t *testing.T) {
	testData := []struct {
		name string

		// Input

		// Expected output
		expect    T
		expectErr error
	}{
		{...},
	}

	for _, testCase := range testData {
		t.Run(testCase.name, func(t *testing.T) {
			// Test logic
			res, err := something(testCase.input)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)
		})
	}
}
```

A few things to consider:

- Test names are written in UpperCamelCase form, to match the case of the test function. This produces a cleaner
  output when running the tests.
- We use the [testify package](https://github.com/stretchr/testify) to make assertions less boilerplate.

Whenever possible, tests should also use the `t.Parallel()` function to run in parallel. This method is not possible
on some layers that are sensitive to deadlocks, such as the `dao` layer. Otherwise, `t.Parallel()` is required by
golang-ci on both tests and subtests.

```go
package layer_test

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSomething(t *testing.T) {
	t.Parallel()

	testData := ...

	for _, testCase := range testData {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// ...
		})
	}
}
```

## Mocking

We use [mockery](https://vektra.github.io/mockery/latest/) to generate mocks. Higher layers like services depends on
interface rather than concrete types. Those interfaces are mocked using a specific command in the makefile:

```shell
make mocks
```

This generates definitions for every configured subpackage.

```go
package layer_test

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSomething(t *testing.T) {
	type fooData struct {
		resp MT
		err  error
	}

	testData := []struct {
		name string

		// Input

		// Mock setup
		fooData *fooData

		// Expected output
		expect    T
		expectErr error
	}{
		{...},
	}

	for _, testCase := range testData {
		t.Run(testCase.name, func(t *testing.T) {
			fooMock := layermocks.NewFooMock(t)

			if testCase.fooData != nil {
				fooMock.On("Method", ...).Return(testCase.fooData.resp, testCase.fooData.err)
			}

			//...

			res, err := something(testCase.input)

			//...

			fooMock.AssertExpectations(t)
		})
	}
}
```
