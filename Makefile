# ================================================================================
# Run tests.
# ================================================================================
test-unit:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.sh'"

test-pkg-js:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.pkg.js.sh'"

test: test-unit test-pkg-js

# ================================================================================
# Lint.
# ================================================================================
lint-go:
	go tool golangci-lint run

lint-node:
	pnpm lint

lint: lint-go lint-node

# ================================================================================
# Format.
# ================================================================================
format-go:
	go mod tidy
	go tool golangci-lint run --fix

format-node:
	pnpm format

format: format-go format-node

# ================================================================================
# Generate.
# ================================================================================
generate-go:
	go generate ./...

generate-node:
	pnpm generate

generate: generate-go generate-node

# ================================================================================
# Local dev.
# ================================================================================
run:
	bash -c "set -m; bash '$(CURDIR)/scripts/run.sh'"

build:
	bash -c "set -m; bash '$(CURDIR)/scripts/build.sh'"

install:
	go mod tidy
	pnpm i --frozen-lockfile
