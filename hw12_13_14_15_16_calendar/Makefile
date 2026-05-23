.PHONY: build run tidy install-lint lint generate

# === Переменные для версионирования ===
RELEASE ?= v0.0.1
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")

# Пути к переменным в internal/version
LDFLAGS := -X github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/version.Release=$(RELEASE) \
           -X github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/version.BuildDate=$(BUILD_DATE) \
           -X github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/version.GitHash=$(GIT_HASH)

# Переменные для бинарников
BINARY_API        = calendar
BINARY_SCHEDULER  = calendar_scheduler
BINARY_SENDER     = calendar_sender

build:
	@echo "Building calendar services..."
	go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_API) ./cmd/calendar
	go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_SCHEDULER) ./cmd/calendar_scheduler
	go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_SENDER) ./cmd/calendar_sender
	@echo "Build complete: 3 binaries in bin/"
	@./bin/calendar --version

run: build
	@echo "Starting calendar service..."
	./bin/calendar --config=configs/calendar.yaml

tidy:
	go mod tidy

install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.63.4

lint: install-lint-deps
	golangci-lint run ./...

generate:
	go generate ./...


# === Migration (main DB) ===
.PHONY: migrate migrate-up migrate-down migrate-status migrate-redo

# Переменные для подключения к основной БД (можно переопределять через ENV)
DB_HOST ?= localhost
DB_PORT ?= 5539
DB_USER ?= intranet
DB_PASSWORD ?= password
DB_NAME ?= calendar
DB_SSLMODE ?= disable

# Формируем DSN из переменных
DSN = "host=$(DB_HOST) port=$(DB_PORT) user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) sslmode=$(DB_SSLMODE)"

# Основная команда: make migrate CMD=up
migrate:
	@echo "Running goose $(CMD) on $(DB_NAME)..."
	@if [ -z "$(CMD)" ]; then \
		echo "Please specify command: make migrate CMD=up|down|status|redo"; \
		exit 1; \
	fi
	goose -dir migrations postgres $(DSN) $(CMD)

# Удобные алиасы для частых команд
migrate-up:
	@$(MAKE) migrate CMD=up

migrate-down:
	@$(MAKE) migrate CMD=down

migrate-status:
	@$(MAKE) migrate CMD=status

migrate-redo:
	@$(MAKE) migrate CMD=redo


# === GOOSE ===
.PHONY: install-goose-deps

GOOSE_VERSION := v3.20.0
GOOSE_BIN   := $(shell go env GOPATH)/bin/goose

install-goose-deps:
	@if [ ! -f "$(GOOSE_BIN)" ]; then \
		echo "goose not found, installing $(GOOSE_VERSION)..."; \
		go install github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION); \
	fi


# === Test DB ===
.PHONY: test test-db-up test-db-down

# Переменные для тестовой БД (можно переопределять через ENV)
TEST_DB_HOST ?= localhost
TEST_DB_PORT ?= 5540
TEST_DB_USER ?= intranet
TEST_DB_PASSWORD ?= password
TEST_DB_NAME ?= calendar_test
TEST_DB_SSLMODE ?= disable

# Формируем DSN для тестовой БД
TEST_DSN = "host=$(TEST_DB_HOST) port=$(TEST_DB_PORT) user=$(TEST_DB_USER) password=$(TEST_DB_PASSWORD) dbname=$(TEST_DB_NAME) sslmode=$(TEST_DB_SSLMODE)"

# Путь к docker compose для тестов (можно переопределить)
TEST_COMPOSE_FILE ?= deployments/docker/calendar/docker compose.test.yml
TEST_CONTAINER_NAME ?= otus_calendar_postgres_test

test-db-up:
	docker compose -f $(TEST_COMPOSE_FILE) up -d
	@echo "Waiting for DB..."
	@until docker exec $(TEST_CONTAINER_NAME) pg_isready -U $(TEST_DB_USER) -d $(TEST_DB_NAME) > /dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "DB is ready!"
	@sleep 1

test-db-down:
	docker compose -f $(TEST_COMPOSE_FILE) down

test: test-db-up install-goose-deps
	@echo "Running migrations on test DB..."
	goose -dir migrations postgres $(TEST_DSN) up
	@echo "Running tests..."
	TEST_DATABASE_URL=$(TEST_DSN) \
		go test -race -count=1 ./... -v
	@$(MAKE) test-db-down


# === Scheduler ===
.PHONY: run-scheduler
run-scheduler:
	@echo "Starting scheduler..."
	./bin/$(BINARY_SCHEDULER) --config=configs/scheduler.yaml


# === Sender ===
.PHONY: run-sender
run-sender:
	@echo "Starting sender..."
	./bin/$(BINARY_SENDER) --config=configs/sender.yaml


# === Docker Management  ===
COMPOSE_DEV := deployments/docker/calendar/docker compose.yml

.PHONY: up
up:
	@echo "Starting development environment..."
	docker compose -f $(COMPOSE_DEV) up -d
	@echo "Services started."

.PHONY: up-build
up-build:
	@echo "Rebuilding with real version info..."
	GIT_HASH=$$(git rev-parse --short HEAD 2>/dev/null || echo "dev") \
	BUILD_DATE=$$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
	docker compose -f $(COMPOSE_DEV) up -d --build
	@echo "Rebuild complete."
	@echo "Services started."

.PHONY: down
down:
	@echo "Stopping containers..."
	docker compose -f $(COMPOSE_DEV) down
	@echo "Containers stopped. Data preserved."

.PHONY: clean
clean:
	@echo "Full cleanup (containers + volumes)..."
	docker compose -f $(COMPOSE_DEV) down -v
	@echo "Everything wiped."

.PHONY: logs
logs:
	docker compose -f $(COMPOSE_DEV) logs -f --tail=100 calendar

.PHONY: logs-sender
logs-sender:
	docker compose -f $(COMPOSE_DEV) logs -f --tail=100 calendar_sender

.PHONY: logs-scheduler
logs-scheduler:
	docker compose -f $(COMPOSE_DEV) logs -f --tail=100 calendar_scheduler


# === Integration Tests ===
.PHONY: integration-tests
integration-tests:
	@echo "Running integration tests..."
	PROJECT_ROOT=$(PWD) go test -v -tags=integration -timeout=5m -count=1 ./tests/integration/...
	@echo "Integration tests complete."

.PHONY: integration-tests-build
integration-tests-build:
	@echo "Building test environment images..."
	docker compose -f deployments/docker/calendar/docker compose.integration.yml build
	@echo "Running integration tests..."
	PROJECT_ROOT=$(PWD) go test -v -tags=integration -timeout=5m -count=1 ./tests/integration/...
	@echo "Integration tests complete."

.PHONY: integration-tests-clean
integration-tests-clean:
	@echo "Cleaning integration test environment..."
	docker compose -f deployments/docker/calendar/docker compose.integration.yml down -v
	@echo "Cleaned."
