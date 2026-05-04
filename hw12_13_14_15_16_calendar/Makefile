.PHONY: build run migrate tidy install-lint lint migrate-up migrate-down migrate-status tidy test-db-up test-db-down test

# === Переменные для версионирования ===
GIT_HASH   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
VERSION    := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.1")

# Убираем кавычки и префикс -ldflags из переменной, чтобы Make не ломал аргументы
LDFLAGS := -X main.release=$(VERSION) -X main.buildDate=$(BUILD_DATE) -X main.gitHash=$(GIT_HASH)

.PHONY: build verify

build:
	@echo "Building calendar service..."	
	go build -ldflags="$(LDFLAGS)" -o bin/calendar ./cmd/calendar
	@./bin/calendar --version

run: build
	@echo "Starting calendar service..."
	./bin/calendar --config=configs/config.yaml

tidy:
	go mod tidy

# LINT_VERSION := v1.63.4
# # Динамически получаем GOPATH. Если не задан, подставит ~/go
# GO_BIN := $(shell go env GOPATH)/bin

install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.63.4

lint: install-lint-deps
	golangci-lint run ./...

# === Migration (main DB) ===
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

# === Test DB ===
# Переменные для тестовой БД (можно переопределять через ENV)
TEST_DB_HOST ?= localhost
TEST_DB_PORT ?= 5540
TEST_DB_USER ?= intranet
TEST_DB_PASSWORD ?= password
TEST_DB_NAME ?= calendar_test
TEST_DB_SSLMODE ?= disable

# Формируем DSN для тестовой БД
TEST_DSN = "host=$(TEST_DB_HOST) port=$(TEST_DB_PORT) user=$(TEST_DB_USER) password=$(TEST_DB_PASSWORD) dbname=$(TEST_DB_NAME) sslmode=$(TEST_DB_SSLMODE)"

# Путь к docker-compose для тестов (можно переопределить)
TEST_COMPOSE_FILE ?= deployments/docker/calendar/docker-compose.test.yml
TEST_CONTAINER_NAME ?= otus_calendar_postgres_test

test-db-up:
	docker-compose -f $(TEST_COMPOSE_FILE) up -d
	@echo "Waiting for DB..."
	@until docker exec $(TEST_CONTAINER_NAME) pg_isready -U $(TEST_DB_USER) -d $(TEST_DB_NAME) > /dev/null 2>&1; do \
		sleep 1; \
	done
	@echo "DB is ready!"

test-db-down:
	docker-compose -f $(TEST_COMPOSE_FILE) down

test: test-db-up
	@echo "Running migrations on test DB..."
	goose -dir migrations postgres $(TEST_DSN) up
	@echo "Running tests..."
	TEST_DATABASE_URL=$(TEST_DSN) \
		go test -race -count=1 ./... -v
	@$(MAKE) test-db-down
