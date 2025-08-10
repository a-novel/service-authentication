#  Run tests.
test:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.sh'"

# Check code quality.
lint:
	go tool golangci-lint run
	npx prettier . --check
	sqlfluff lint

# Generate mocked interfaces for Go tests.
mocks:
	rm -rf `find . -type d -name mocks`
	go run github.com/vektra/mockery/v3@v3.5.2

# Reformat code so it passes the code style lint checks.
format:
	go mod tidy
	go tool golangci-lint run --fix
	npx prettier . --write
	sqlfluff fix

# Lint OpenAPI specs.
openapi-lint:
	npx @redocly/cli lint ./docs/api.yaml

# Generate Go code.
go-generate:
	go generate ./...

mjml-lint:
	for i in `find ./internal/lib/mails -name "*.mjml" -type f`; do \
		npx --yes --package=mjml@next mjml --validate $$i; \
	done

mjml-generate:
	for i in `find ./internal/lib/mails -name "*.mjml" -type f`; do \
		npx --yes --package=mjml@next mjml $$i -o $${i%.*}.html; \
	done

run-infra:
	podman compose -p "${APP_NAME}" -f "${PWD}/build/podman-compose.yaml" up -d --build --pull-always

run-infra-down:
	podman compose -p "${APP_NAME}" -f "${PWD}/build/podman-compose.yaml" down

# Run the API
run-api:
	bash -c "set -m; bash '$(CURDIR)/scripts/run.sh'"

install:
	pipx install sqlfluff
