.PHONY: generate check-env migrate migrate-down run test test-db lint

generate:
	CGO_ENABLED=0 go tool oapi-codegen -generate types,chi-server -include-operation-ids createTrip,getTrip,finishTrip,health,ready -package api -o internal/generated/api.gen.go contracts/openapi/trip-service.openapi.yaml

check-env:
	@test -f .env || { echo "Нет .env. Выполните tripgoctl environment start из корня проекта." >&2; exit 1; }
	@set -eu; set -a; . ./.env; set +a; test -n "$${DATABASE_URL:-}" || { echo "В .env не задан DATABASE_URL." >&2; exit 1; }

migrate: check-env
	@set -eu; set -a; . ./.env; set +a; CGO_ENABLED=0 go tool goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down: check-env
	@set -eu; set -a; . ./.env; set +a; CGO_ENABLED=0 go tool goose -dir migrations postgres "$$DATABASE_URL" down

run: check-env
	@set -eu; set -a; . ./.env.example; . ./.env; set +a; CGO_ENABLED=0 go run ./cmd/trip-service

test:
	CGO_ENABLED=1 go test -race ./...

test-db: check-env
	@set -eu; set -a; . ./.env; set +a; CGO_ENABLED=1 go test -race ./... -count=1

lint:
	CGO_ENABLED=0 go vet ./...
