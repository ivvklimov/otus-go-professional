package config

import "fmt"

type Config struct {
	Logger  LoggerConfig  `yaml:"logger"`
	Server  ServerConfig  `yaml:"server"`
	Storage StorageConfig `yaml:"storage"`
	DB      DBConfig      `yaml:"db"`
	GRPC    GRPCConfig    `yaml:"grpc"`
}

type LoggerConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type StorageConfig struct {
	Type string `yaml:"type"`
}

type DBConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	DBName       string `yaml:"dbname"`
	SSLMode      string `yaml:"sslmode"`
	MaxOpenConns int    `yaml:"max_open_conns"`
}

type GRPCConfig struct {
	Port string `yaml:"port"`
}

// DSN возвращает готовую строку подключения для lib/pq.
func (c *DBConfig) DSN() string {
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, sslMode,
	)
}
