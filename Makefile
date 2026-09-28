.PHONY: generate migrate migrate-down run test lint

generate:
	CGO_ENABLED=0 go tool oapi-codegen -generate types,chi-server -package api -o internal/generated/api.gen.go contracts/openapi/trip-service.openapi.yaml

migrate:
	@set -a; . ./.env; set +a; CGO_ENABLED=0 go tool goose -dir migrations postgres "$$DATABASE_URL" up

migrate-down:
	@set -a; . ./.env; set +a; CGO_ENABLED=0 go tool goose -dir migrations postgres "$$DATABASE_URL" down

run:
	@set -a; . ./.env.example; . ./.env; set +a; CGO_ENABLED=0 go run ./cmd/trip-service

test:
	CGO_ENABLED=0 go test ./...

lint:
	CGO_ENABLED=0 go vet ./...
