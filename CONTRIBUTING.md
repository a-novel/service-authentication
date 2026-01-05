# Contributing

## Prerequisites

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

## Bootstrap

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

## Project structure

- `/builds`: Build files for the containerized exports. Must be imported through the registry (see Readme).
- `/cmd`: Uncompiled commands of the service, used by the builds.
- `/internal`: this is where the core of the service lies. Compiled commands and containers use this code.
  It contains the hexagonal architecture layers.
  - `/config`: Configuration for the service, where editable values are loaded from various sources (yaml, env, etc.)
  - `/dao`: Data Access Objects, responsible for interacting with the data sources.
  - `/handlers`: Interface between the outside world and the application logic. Takes serialized requests, parse them
    for internal usage and serializes the response.
  - `/lib`: Shared utilities and helper functions used across the application.
  - `/models`: Defines external models used by the application (database migrations, templates, etc.)
  - `/services`: Core business logic of the application. Implements use cases and orchestrates interactions
    between different components.
- `/pkg`: Exported packages to interact with the service programmatically. Imported through dedicated package
  managers (see Readme)
- `/scripts`: Bash utils used during development and build.

## Development commands

We use Make to run commands during development. Those are the most useful commands,
you can look at the Makefile directly for the more detailed set.

Run tests

```bash
make test
```

Format / lint code

```bash
make lint
make format # to fix
```

When modifying some parts of the code, you'll likely want to make sure all
the generated files are up to date. You can do so by running:

```bash
make generate
```
