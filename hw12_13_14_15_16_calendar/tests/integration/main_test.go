//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const (
	// Базовый URL тестового API (порт 8889, чтобы не конфликтовал с основным 8888)
	testAPIURL = "http://localhost:8889"

	// Таймауты для ожидания готовности сервисов
	waitTimeout  = 60 * time.Second // Увеличено для медленных окружений (CI)
	waitInterval = 1 * time.Second  // Интервал между попытками
)

// getComposeFile возвращает абсолютный путь к файлу docker-compose для тестов.
// Приоритет: переменная окружения PROJECT_ROOT > относительный путь.
func getComposeFile() string {
	if root := os.Getenv("PROJECT_ROOT"); root != "" {
		path := filepath.Join(root, "deployments/docker/calendar/docker-compose.integration.yml")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	// Фоллбэк: относительный путь от папки tests/integration/
	return "../../../deployments/docker/calendar/docker-compose.integration.yml"
}

// TestMain — точка входа для пакета интеграционных тестов.
// Жизненный цикл:
// 1. Поднимает изолированное окружение (БД, RabbitMQ, миграции, сервисы).
// 2. Ждёт готовности API (health check).
// 3. Запускает тесты (m.Run()).
// 4. Гарантированно очищает окружение (даже при панике).
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	composeFile := getComposeFile()
	fmt.Printf("Starting test environment (compose: %s)...\n", composeFile)

	// Поднимаем окружение
	if err := runCompose(ctx, composeFile, "up", "-d", "--build"); err != nil {
		fmt.Printf("Failed to start environment: %v\n", err)
		os.Exit(1)
	}

	// Гарантированная очистка при выходе (даже если тесты упадут с паникой)
	defer func() {
		fmt.Println("Cleaning up test environment...")
		_ = runCompose(context.Background(), composeFile, "down", "-v")
	}()

	// Ждём готовности API
	fmt.Println("Waiting for API to be ready...")
	if err := waitForAPI(ctx, testAPIURL, waitTimeout, waitInterval); err != nil {
		fmt.Printf("API not ready: %v\n", err)
		// Выводим логи упавшего сервиса для отладки
		_ = runCompose(context.Background(), composeFile, "logs", "calendar-test")
		os.Exit(1)
	}
	fmt.Println("API is ready")

	// Запускаем тесты
	code := m.Run()
	os.Exit(code)
}

// runCompose — обёртка над docker-compose с поддержкой context.
func runCompose(ctx context.Context, composeFile string, args ...string) error {
	cmdArgs := append([]string{"-f", composeFile}, args...)
	cmd := exec.CommandContext(ctx, "docker-compose", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// waitForAPI — polling до тех пор, пока сервис не начнёт отвечать.
// Считает сервис «готовым», если он возвращает любой код 200–499.
// Это позволяет игнорировать 400/404 (валидные ответы бизнес-логики)
// и реагировать только на реальные ошибки сервера (5xx) или таймауты соединения.
func waitForAPI(ctx context.Context, baseURL string, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	healthURL := baseURL + "/api/v1/calendar/events/day?date=2026-01-01T00:00:00Z"

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		resp, err := client.Get(healthURL)
		if err == nil {
			// 2xx — успех, 4xx — сервер жив (ошибка клиента), 5xx — сервер упал
			if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusInternalServerError {
				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				return nil
			}
			_ = resp.Body.Close()
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("timeout waiting for %s", healthURL)
}

// httpPostJSON — хелпер для отправки JSON-запросов с правильными заголовками.
func httpPostJSON(url string, body interface{}) (*http.Response, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, fmt.Errorf("encode json: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return http.DefaultClient.Do(req)
}
