.PHONY: run test test-integration lint tidy migrate-up migrate-down migrate-supabase-up migrate-supabase-down migrate-supabase-force

run:
	go run ./cmd/api

test:
	go test ./...

test-integration:
	docker compose up -d postgres
	docker compose --profile tools run --rm migrate -path=/migrations -database="postgres://multibook:multibook@postgres:5432/multibook?sslmode=disable" up
	INTEGRATION_DATABASE_URL="postgres://multibook:multibook@localhost:5432/multibook?sslmode=disable" go test -tags=integration ./internal/integration/...

lint:
	go vet ./...

tidy:
	go mod tidy

migrate-up:
	docker compose up -d postgres
	docker compose --profile tools run --rm migrate -path=/migrations -database="postgres://multibook:multibook@postgres:5432/multibook?sslmode=disable" up

migrate-down:
	docker compose up -d postgres
	docker compose --profile tools run --rm migrate -path=/migrations -database="postgres://multibook:multibook@postgres:5432/multibook?sslmode=disable" down 1

migrate-supabase-up:
	docker run --rm --env-file .env -v "$(CURDIR)/migrations:/migrations:ro" --entrypoint /bin/sh migrate/migrate:v4.18.3 -c 'migrate -path=/migrations -database="$$DATABASE_URL" up'

migrate-supabase-down:
	docker run --rm --env-file .env -v "$(CURDIR)/migrations:/migrations:ro" --entrypoint /bin/sh migrate/migrate:v4.18.3 -c 'migrate -path=/migrations -database="$$DATABASE_URL" down 1'

migrate-supabase-force:
	@test -n "$(VERSION)" || (echo "VERSION is required, e.g. make migrate-supabase-force VERSION=3"; exit 1)
	docker run --rm --env-file .env -v "$(CURDIR)/migrations:/migrations:ro" --entrypoint /bin/sh migrate/migrate:v4.18.3 -c 'migrate -path=/migrations -database="$$DATABASE_URL" force $(VERSION)'
