SHELL := /bin/bash
LINT_VERSION := v2.15.0
LINT := $(CURDIR)/.tools/bin/golangci-lint

.PHONY: generate check-env migrate migrate-down run test test-integration test-db lint format tools

generate:
	go tool oapi-codegen -generate types,chi-server -include-operation-ids createTrip,getTrip,finishTrip,health,ready -package api -o internal/generated/api.gen.go contracts/openapi/trip-service.openapi.yaml
	go generate ./internal/...
	@find internal -name 'mocks_test.go' -o -name 'api.gen.go' | xargs sed -i '/^[[:space:]]*\/\//{/^\/\/ Code generated/!d;}'
	gofmt -w internal/generated $$(find internal -name mocks_test.go)

check-env:
	@test -f .env || { echo 'Run tripgoctl environment start first.' >&2; exit 1; }

migrate: check-env
	@set -eu; set -a; . .env; set +a; go tool goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down: check-env
	@set -eu; set -a; . .env; set +a; go tool goose -dir migrations postgres "$$DATABASE_URL" down

run: check-env
	@set -eu; set -a; . .env.example; . .env; set +a; go run ./cmd/trip-service

test:
	go test -race -count=2 -coverprofile=coverage.out -coverpkg=./internal/domain/...,./internal/usecase/... ./...
	go tool cover -func=coverage.out

test-integration:
	@set -eu; set -a; if [ -f .env ]; then . .env; fi; set +a; : "$${DATABASE_URL:?Run tripgoctl environment start and make migrate first.}"; go test -race -count=2 -tags=integration ./internal/repository/...

test-db: test-integration

tools: $(LINT)

$(LINT):
	mkdir -p .tools/bin
	GOBIN=$(CURDIR)/.tools/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(LINT_VERSION)

lint: $(LINT)
	$(LINT) run

format: $(LINT)
	$(LINT) fmt
