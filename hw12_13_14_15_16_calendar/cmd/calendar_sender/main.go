package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/broker"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/config"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/logger"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/version"
)

// SenderConfig — структура конфигурации для рассыльщика.
type SenderConfig struct {
	RMQURL string          `yaml:"rmq_url"`
	Queue  string          `yaml:"queue"`
	DB     config.Database `yaml:"db"`
}

// Глобальные переменные для флагов.
var (
	configFile  string
	showVersion bool
)

// init регистрирует флаги (выполняется до main).
func init() {
	flag.StringVar(&configFile, "config", "configs/sender.yaml", "Path to configuration file")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
}

// fatalf выводит ошибку через log и завершает программу (аналог log.Fatalf).
func fatalf(log *logger.Logger, format string, args ...interface{}) {
	log.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

func main() {
	flag.Parse()

	// 1. Проверка версии - до загрузки конфига
	// Работает даже если конфиг не передан или не валиден
	if showVersion || flag.Arg(0) == "version" {
		version.Print()
		return
	}

	// 2. Загружаем базовый конфиг из файла (ConfigMap в K8s)
	cfg, err := config.Load[SenderConfig](configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 3. Переопределяем секреты из ENV (K8s Secrets)
	applySecrets(cfg)

	// 4. Инициализация логгера
	log := logger.New("info")
	log.Info(fmt.Sprintf("Starting Calendar Sender (version: %s)", version.String()))

	// 5. Контекст с graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// 6. Подключение к БД (для обновления notify_delivered_at)
	dsn := cfg.DB.DSN()
	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		fatalf(log, "failed to connect to PostgreSQL: %v", err)
	}

	// Применяем настройки пула из конфига (с дефолтами)
	cfg.DB.ApplyPoolSettings(db)

	defer db.Close()
	log.Info("Connected to PostgreSQL")

	// 7. Подключение к RabbitMQ
	b := broker.NewAMQPBroker(cfg.RMQURL)
	if err := b.Connect(ctx); err != nil {
		fatalf(log, "failed to connect to RabbitMQ: %v", err)
	}
	defer b.Close()

	if err := b.DeclareQueue(ctx, cfg.Queue); err != nil {
		fatalf(log, "failed to declare queue: %v", err)
	}

	// 8. Запуск потребителя
	msgs, err := b.Consume(ctx, cfg.Queue)
	if err != nil {
		fatalf(log, "failed to start consuming: %v", err)
	}

	log.Info("Waiting for messages...")

	// 9. Цикл обработки сообщений с поддержкой graceful shutdown
	for {
		select {
		case <-ctx.Done():
			log.Info("Shutdown signal received, stopping consumer...")
			return
		case msg, ok := <-msgs:
			if !ok {
				log.Info("Message channel closed")
				return
			}

			var notification storage.Notification
			if err := json.Unmarshal(msg.Body, &notification); err != nil {
				log.Error("failed to unmarshal: " + err.Error())
				_ = msg.NACK()
				continue
			}

			logMsg := fmt.Sprintf(
				"✉  NOTIFICATION DELIVERED | ID: %s | Title: %s | Date: %s | Owner: %d",
				notification.EventID,
				notification.Title,
				notification.DateStart.Format(time.RFC3339),
				notification.OwnerID,
			)
			log.Info(logMsg)

			if err := markDelivered(ctx, db, notification.EventID); err != nil {
				log.Error("failed to update delivery status: " + err.Error())
				_ = msg.NACK()
				continue
			}

			if err := msg.ACK(); err != nil {
				log.Error("failed to ACK: " + err.Error())
			}
		}
	}
}

// applySecrets переопределяет чувствительные настройки из ENV.
func applySecrets(cfg *SenderConfig) {
	if pass := os.Getenv("DB_PASSWORD"); pass != "" {
		cfg.DB.Password = pass
	}
	if url := os.Getenv("RMQ_URL"); url != "" {
		cfg.RMQURL = url
	}
}

// markDelivered обновляет notify_delivered_at для события.
func markDelivered(ctx context.Context, db *sqlx.DB, eventID string) error {
	query := `UPDATE events SET notify_delivered_at = NOW() WHERE id = $1`
	_, err := db.ExecContext(ctx, query, eventID)
	return err
}
