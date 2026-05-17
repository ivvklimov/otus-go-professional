package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Load загружает конфигурацию из файла по указанному пути.
// Поддерживает переопределение некоторых полей через ENV-переменные.
func Load(path string) (*Config, error) {
	// 1. Читаем файл
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file %s: %w", path, err)
	}

	// 2. Парсим YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// 3. Применяем ENV-переменные (приоритет выше, чем у файла)
	// Это стандарт для 12-Factor App в Docker/K8s

	// Logger
	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		cfg.Logger.Level = envLevel
	}

	// Server
	if envPort := os.Getenv("SERVER_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			cfg.Server.Port = p
		}
	}

	// Database (поддержка переопределения параметров подключения)
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

	// 4. Валидация (Fail-fast: лучше упасть сразу, чем работать с мусором)

	// Server validation
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return nil, fmt.Errorf("invalid server port: %d", cfg.Server.Port)
	}
	if cfg.Logger.Level == "" {
		cfg.Logger.Level = "info" // Дефолтное значение
	}

	// Storage type default
	if cfg.Storage.Type == "" {
		cfg.Storage.Type = "memory"
	}

	// DB validation (только если выбрано SQL хранилище)
	if cfg.Storage.Type == "sql" {
		if cfg.DB.Host == "" {
			return nil, fmt.Errorf("db.host is required for sql storage")
		}
		if cfg.DB.Port <= 0 || cfg.DB.Port > 65535 {
			return nil, fmt.Errorf("db.port must be valid (1-65535)")
		}
		if cfg.DB.User == "" {
			return nil, fmt.Errorf("db.user is required for sql storage")
		}
		if cfg.DB.DBName == "" {
			return nil, fmt.Errorf("db.dbname is required for sql storage")
		}
		// Password может быть пустым (не всегда требуется в локальной БД)
	}

	return &cfg, nil
}
