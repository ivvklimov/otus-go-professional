package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Load загружает конфигурацию из YAML-файла в любую структуру.
// Универсальная функция для всех сервисов: calendar, scheduler, sender.
func Load[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}

	var cfg T
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// ApplyAPIEnvOverrides применяет переменные окружения к конфигурации API.
// Вызывайте после config.Load[Config]() в cmd/calendar/main.go.
func ApplyAPIEnvOverrides(cfg *Config) {
	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		cfg.Logger.Level = envLevel
	}
	if envPort := os.Getenv("SERVER_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			cfg.Server.Port = p
		}
	}
	if env := os.Getenv("DB_HOST"); env != "" {
		cfg.DB.Host = env
	}
	if env := os.Getenv("DB_PORT"); env != "" {
		if p, err := strconv.Atoi(env); err == nil {
			cfg.DB.Port = p
		}
	}
	if env := os.Getenv("DB_USER"); env != "" {
		cfg.DB.User = env
	}
	if env := os.Getenv("DB_PASSWORD"); env != "" {
		cfg.DB.Password = env
	}
	if env := os.Getenv("DB_NAME"); env != "" {
		cfg.DB.DBName = env
	}
	if env := os.Getenv("DB_SSLMODE"); env != "" {
		cfg.DB.SSLMode = env
	}
}

// ValidateAPI проверяет корректность конфигурации API.
func ValidateAPI(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}
	if cfg.Logger.Level == "" {
		cfg.Logger.Level = "info"
	}
	if cfg.Storage.Type == "" {
		cfg.Storage.Type = "memory"
	}

	if cfg.Storage.Type == "sql" {
		if cfg.DB.Host == "" {
			return fmt.Errorf("db.host is required for sql storage")
		}
		if cfg.DB.Port <= 0 || cfg.DB.Port > 65535 {
			return fmt.Errorf("db.port must be valid (1-65535)")
		}
		if cfg.DB.User == "" {
			return fmt.Errorf("db.user is required for sql storage")
		}
		if cfg.DB.DBName == "" {
			return fmt.Errorf("db.dbname is required for sql storage")
		}
	}
	return nil
}
