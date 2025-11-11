#  Run tests.
test-unit:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.sh'"

test-integration-js:
	bash -c "set -m; bash '$(CURDIR)/scripts/test-integration-js.sh'"

test: test-unit test-integration-js

# Check code quality.
lint:
	go tool golangci-lint run
	pnpm lint

# Reformat code so it passes the code style lint checks.
format:
	go mod tidy
	go tool golangci-lint run --fix
	pnpm format

# Generate Go code.
generate-go:
	go generate ./...

generate: generate-go
	pnpm generate

# Run the rest API
run:
	bash -c "set -m; bash '$(CURDIR)/scripts/run.sh'"

build:
	bash -c "set -m; bash '$(CURDIR)/scripts/build.sh'"

install:
	go mod tidy
	pnpm i --frozen-lockfile
