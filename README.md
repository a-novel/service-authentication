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

## Prerequisites

- [Go](https://go.dev/doc/install)
- [Node.js](https://nodejs.org/en/download)
  - [pnpm](https://pnpm.io/installation)
- [Podman](https://podman.io/docs/installation)
- [Direnv](https://direnv.net/)
- Make
  - `sudo apt-get install build-essential` (apt)
  - `sudo pacman -S make` (arch)
  - `https://gnuwin32.sourceforge.net/packages/make.htm` (macOS)
  - [Make for Windows](https://gnuwin32.sourceforge.net/packages/make.htm)

## Installation

Install dependencies

```bash
make install
```

Create env file

```bash
cp .envrc.template .envrc
```

Ask an admin to get the actual values for the placeholders in the new `.envrc` file (indicated by surrounding `[]`
brackets).

## Run locally

### As Rest API

```bash
make run
```

Interact with the server (in a different directory):

```bash
curl http://localhost:4011/ping
# pong
curl http://localhost:4011/healthcheck
# {""client:postgres":{"status":"up"},...
```

> Note: the `run` script handles graceful shutdown and cleanup of the server resources. You can quit the server by 
> killing it with Ctrl+C / Cmd+C, however beware this will not terminate immediately, and instead trigger the cleanup 
> script.

### As Containers

You can build local version of the containers using
```bash
make build
```

You can then use the `:local` tag and the official image handler 
(eg: `ghcr.io/a-novel/service-authentication/standalone:local`)

## Contribute

Run tests
```bash
make test
```

Make sure the code complies to our guidelines
```bash
make lint # make format
```

Keep the code up-to-date
```bash
make generate
```
