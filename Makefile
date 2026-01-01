.PHONY: build run test clean migrate-up migrate-down migrate-create dev

build:
	go build -o bin/api ./cmd/api

run: build
	./bin/api

dev:
	air

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

deps:
	go mod download
	go mod tidy

lint:
	golangci-lint run

migrate-up: build
	./bin/api -migrate-up

migrate-down: build
	./bin/api -migrate-down

migrate-create:
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=migration_name"; exit 1; fi
	@echo "Creating migration: $(name)"
	@touch migrations/$$(printf "%06d" $$(($$(ls migrations/*.up.sql 2>/dev/null | wc -l) + 1)))_$(name).up.sql
	@touch migrations/$$(printf "%06d" $$(($$(ls migrations/*.up.sql 2>/dev/null | wc -l))))_$(name).down.sql
	@echo "Created migrations for: $(name)"

fmt:
	go fmt ./...
	goimports -w .

install-migrate:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
