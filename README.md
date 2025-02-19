![Authentication Service](./docs/assets/service%20authentication%20banner.png)

[![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/agora_ecrivains)](https://twitter.com/agora_ecrivains)
[![Discord](https://img.shields.io/discord/1315240114691248138?logo=discord)](https://discord.gg/D7rqySm8)

<hr />

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/a-novel/authentication)
![GitHub repo file or directory count](https://img.shields.io/github/directory-file-count/a-novel/authentication)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/a-novel/authentication)

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/a-novel/authentication/main.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/a-novel/authentication)](https://goreportcard.com/report/github.com/a-novel/authentication)
[![codecov](https://codecov.io/gh/a-novel/authentication/graph/badge.svg?token=cnSwTJ2q4n)](https://codecov.io/gh/a-novel/authentication)

![Coverage graph](https://codecov.io/gh/a-novel/authentication/graphs/sunburst.svg?token=cnSwTJ2q4n)

<hr />

This is a quickstart document to test the project locally.

You can find the API documentation on the [repository github page](https://a-novel.github.io/authentication/).

Want to contribute? Check the [contribution guidelines](CONTRIBUTING.md).

# Run locally

## Pre-requisites

- [Golang](https://go.dev/doc/install)
- [Podman](https://podman.io/docs/installation)
- Make
  ```bash
  # Debian / Ubuntu
  sudo apt-get install build-essential
  
  # macOS
  brew install make
  ```
  For Windows, you can use [Make for Windows](https://gnuwin32.sourceforge.net/packages/make.htm)

## Setup environment

Create a `.envrc` file from the template:

```bash
cp .envrc.template .envrc
```

Then fill the missing secret variables. Once your file is ready:

```bash
source .envrc
```

> You may use tools such as [direnv](https://direnv.net/), otherwise you'll need to source the env file on each new
> terminal session.

## Generate keys

You need to do this at least once, to have a set of keys ready to use for authentication.

> It is recommended to run this regularly, otherwise keys will expire and authentication
> will fail.

```bash
make rotate_keys
# 9:07PM INF key generated app=authentication job=rotate-keys key_id=... usage=auth
# 9:07PM INF no key generated app=authentication job=rotate-keys usage=refresh
# 9:07PM INF rotation done app=authentication failed_keys=0 generated_keys=1 job=rotate-keys total_keys=2
```

## Et Voilà!

```bash
make api
# 3:09PM INF starting application... app=authentication
# 3:09PM INF application started! address=:4001 app=authentication
```
