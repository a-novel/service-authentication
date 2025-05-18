# Run tests.
test:
	bash -c "set -m; bash '$(CURDIR)/scripts/test.sh'"

# Check code quality.
lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run
	npx prettier . --check
	sqlfluff lint

# Generate mocked interfaces for Go tests.
mocks:
	rm -rf `find . -type d -name mocks`
	go run github.com/vektra/mockery/v3@v3.2.5

# Reformat code so it passes the code style lint checks.
format:
	go mod tidy
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run --fix
	npx prettier . --write
	sqlfluff fix

# Lint OpenAPI specs.
openapi-lint:
	npx @redocly/cli lint ./docs/api.yaml

# Generate OpenAPI docs.
openapi-generate:
	go generate ./...

mjml-lint:
	for i in `find ./config/mails -name "*.mjml" -type f`; do \
		npx mjml --validate $$i; \
	done

mjml-generate:
	for i in `find ./config/mails -name "*.mjml" -type f`; do \
		npx mjml $$i -o $${i%.*}.html; \
	done

# Execute the keys rotation job locally.
rotate_keys:
	bash -c "set -m; bash '$(CURDIR)/scripts/run_rotate_keys.sh'"

# Run the API
api:
	bash -c "set -m; bash '$(CURDIR)/scripts/run.sh'"

install:
	pipx install sqlfluff

.PHONY: api
