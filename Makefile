#  Run tests.
test:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.sh'"

# Check code quality.
lint:
	go tool golangci-lint run
	pnpm lint

# Reformat code so it passes the code style lint checks.
format:
	go mod tidy
	go tool golangci-lint run --fix
	pnpm format

# Lint OpenAPI specs.
openapi-lint:
	pnpm lint:openapi

# Generate Go code.
go-generate:
	go generate ./...

run-infra:
	podman compose -p "${APP_NAME}" -f "${PWD}/build/podman-compose.yaml" up -d --build --pull-always

run-infra-down:
	podman compose -p "${APP_NAME}" -f "${PWD}/build/podman-compose.yaml" down

# Run the API
run-api:
	bash -c "set -m; bash '$(CURDIR)/scripts/run.sh'"
