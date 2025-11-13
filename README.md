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

## Import in other projects

### Go package

You will also need [service JSON-Keys](https://github.com/a-novel/service-json-keys).

```bash
go get -u github.com/a-novel/service-authentication
```

```go
package main

import (
	"context"

	"github.com/go-chi/chi/v5"

	authpkg "github.com/a-novel/service-authentication/pkg"
	jkpkg "github.com/a-novel/service-json-keys/v2/pkg"
)

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

	jsonKeysClient, _ := jkpkg.NewClient(ctx, "<service-json-keys-url>")
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

### Javascript package

The client rest api is exposed through a javascript package. It uses native fetch api, and exposes every type through
[zod](https://github.com/colinhacks/zod).

> ⚠️ **Warning**: Even though the package is public, GitHub registry requires you to have a Personal Access Token
> with `repo` and `read:packages` scopes to pull it in your project. See
> [this issue](https://github.com/orgs/community/discussions/23386#discussioncomment-3240193) for more information.

Make sure you have a `.npmrc` with the following content (in your project or in your home directory):

```ini
@a-novel:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${YOUR_PERSONAL_ACCESS_TOKEN}
```

Then, install the package using pnpm:

```bash
# pnpm config set auto-install-peers true
#  Or
# pnpm config set auto-install-peers true --location project
pnpm add @a-novel/service-authentication-rest
```

Usage

```typescript
import { AuthenticationApi, tokenCreateAnon } from "@a-novel/service-authentication-rest";

// API instance can be shared.
const api = new AuthenticationApi("<base_api_url>");

const token = await tokenCreateAnon(api);
```

## Development

### Installation

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

### Run locally

#### As Rest API

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

#### As Containers

You can build local version of the containers using

```bash
make build
```

You can then use the `:local` tag and the official image handler
(eg: `ghcr.io/a-novel/service-authentication/standalone:local`)

### Contribute

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
