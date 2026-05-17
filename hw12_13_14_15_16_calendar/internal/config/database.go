package config

import (
	"fmt"
	"time"
)

// Database содержит настройки подключения к PostgreSQL.
type Database struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`

	// Настройки пула соединений (опционально, используются при инициализации sqlx.DB)
	MaxOpenConns    int `yaml:"max_open_conns"`    // default: 10
	MaxIdleConns    int `yaml:"max_idle_conns"`    // default: 5
	ConnMaxLifetime int `yaml:"conn_max_lifetime"` // default: 30 (minutes)
}

// DSN возвращает строку подключения в формате для lib/pq.
// Не включает настройки пула — они применяются отдельно через методы *sqlx.DB.
func (db *Database) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		db.Host, db.Port, db.User, db.Password, db.DBName, db.SSLMode)
}

// ApplyPoolSettings применяет настройки пула соединений к *sqlx.DB.
// Использует дефолтные значения, если в конфиге указаны нули.
// Вызывать после успешного подключения к БД.
func (db *Database) ApplyPoolSettings(sqlDB interface {
	SetMaxOpenConns(int)
	SetMaxIdleConns(int)
	SetConnMaxLifetime(time.Duration)
}) {
	maxOpen := db.MaxOpenConns
	if maxOpen == 0 {
		maxOpen = 10
	}

	maxIdle := db.MaxIdleConns
	if maxIdle == 0 {
		maxIdle = 5
	}

	lifetime := db.ConnMaxLifetime
	if lifetime == 0 {
		lifetime = 30 // minutes
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(time.Duration(lifetime) * time.Minute)
}
