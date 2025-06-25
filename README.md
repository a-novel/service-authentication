# Authentication service

[![X (formerly Twitter) Follow](https://img.shields.io/twitter/follow/agora_ecrivains)](https://twitter.com/agora_ecrivains)
[![Discord](https://img.shields.io/discord/1315240114691248138?logo=discord)](https://discord.gg/rp4Qr8cA)

<hr />

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/a-novel/service-authentication)
![GitHub repo file or directory count](https://img.shields.io/github/directory-file-count/a-novel/service-authentication)
![GitHub code size in bytes](https://img.shields.io/github/languages/code-size/a-novel/service-authentication)

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/a-novel/service-authentication/main.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/a-novel/service-authentication)](https://goreportcard.com/report/github.com/a-novel/service-authentication)
[![codecov](https://codecov.io/gh/a-novel/service-authentication/graph/badge.svg?token=cnSwTJ2q4n)](https://codecov.io/gh/a-novel/service-authentication)

![Coverage graph](https://codecov.io/gh/a-novel/service-authentication/graphs/sunburst.svg?token=cnSwTJ2q4n)

<hr />

This is a quickstart document to test the project locally.

You can find the API documentation on the [repository GitHub page](https://a-novel.github.io/authentication/).

Want to contribute? Check the [contribution guidelines](CONTRIBUTING.md).

# Use in a project

You can import this application as a docker image. Below is an example using
[podman compose](https://docs.podman.io/en/latest/markdown/podman-compose.1.html).

```yaml
services:
  postgres-authentication:
    image: docker.io/library/postgres:17
    networks:
      - api
    environment:
      POSTGRES_PASSWORD: postgres
      POSTGRES_USER: postgres
      POSTGRES_DB: authentication
      POSTGRES_HOST_AUTH_METHOD: scram-sha-256
      POSTGRES_INITDB_ARGS: --auth=scram-sha-256
    volumes:
      - postgres-auth-data:/var/lib/postgresql/data/

  # Runs the secret key rotation on every launch.
  # Keys are smartly rotated, meaning new keys are generated only when necessary
  # (eg: keys missing or last generated version is too old).
  # The container will exit by itself when the job is done.
  authentication-rotate-keys-job:
    image: ghcr.io/a-novel/service-authentication/jobs/rotatekeys:v0
    depends_on:
      - postgres-authentication
    environment:
      ENV: local
      APP_NAME: authentication-service-rotate-keys-job
      DSN: postgres://postgres:postgres@postgres-authentication:5432/authentication?sslmode=disable
      # Dummy key used only for local environment. Consider using a secure, private key in production.
      # Note it MUST match the one used in the authentication service.
      MASTER_KEY: fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca
    networks:
      - api

  authentication-service:
    image: ghcr.io/a-novel/service-authentication/api:v0
    depends_on:
      - postgres-authentication
    ports:
      # Expose the service on port 4001 on the local machine.
      - "4001:8080"
    environment:
      PORT: 8080
      ENV: local
      APP_NAME: authentication-service
      DSN: postgres://postgres:postgres@postgres-authentication:5432/authentication?sslmode=disable
      # Dummy key used only for local environment. Consider using a secure, private key in production.
      # Note it MUST match the one used in the authentication keys rotation job.
      MASTER_KEY: fec0681a2f57242211c559ca347721766f8a3acd8ed2e63b36b3768051c702ca
      # In sandbox mode, mails are logged in the server logs rather than being sent. Alternatively, you need to provide
      # a valid SMTP server configuration.
      SMTP_SANDBOX: true
      # SMTP_PASSWORD: your_smtp_password
      # SMTP_SENDER: noreply@agoradesecrivains.com
      # SMTP_DOMAIN: smtp-relay.gmail.com
      # SMTP_ADDRESS: smtp-relay.gmail.com:587
      AUTH_PLATFORM_URL_UPDATE_EMAIL: http://localhost:4001/update-email
      AUTH_PLATFORM_URL_UPDATE_PASSWORD: http://localhost:4001/update-password
      AUTH_PLATFORM_URL_REGISTER: http://localhost:4001/register
      # Used for tracing purposes, can be omitted.
      # SENTRY_DSN: [your_sentry_dsn]
      # SERVER_NAME: authentication-service-prod
      # RELEASE: v0.1.2
      # ENV: production
      # Set the following if you want to debug the service locally.
      # DEBUG: true
    networks:
      - api

networks:
  api: {}

volumes:
  postgres-auth-data:
```

Available tags includes:

- `latest`: latest versioned image
- `vx`: versioned images, pointing to a specific version. Partial versions are supported. When provided, the
  latest subversion is used.\
  examples: `v0`, `v0.1`, `v0.1.2`
- `branch`: get the latest version pushed to a branch. Any valid branch name can be used.\
  examples: `master`, `fix/something`

# Run locally

## Pre-requisites

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

Install the external dependencies:

```bash
make install
```

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
