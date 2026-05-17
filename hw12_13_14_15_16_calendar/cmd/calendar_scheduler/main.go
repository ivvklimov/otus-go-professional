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

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/broker"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/config"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/logger"
	sqlstorage "github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage/sql"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/version"
)

// SchedulerConfig - структура конфигурации для планировщика.
type SchedulerConfig struct {
	RMQURL          string          `yaml:"rmq_url"`
	Exchange        string          `yaml:"exchange"`
	Queue           string          `yaml:"queue"`
	RoutingKey      string          `yaml:"routing_key"`
	PollInterval    time.Duration   `yaml:"poll_interval"`
	CleanupInterval time.Duration   `yaml:"cleanup_interval"`
	DB              config.Database `yaml:"db"`
}

// Глобальные переменные для флагов.
var (
	configFile  string
	showVersion bool
)

// init регистрирует флаги.
func init() {
	flag.StringVar(&configFile, "config", "configs/scheduler.yaml", "Path to configuration file")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
}

// fatalf выводит ошибку через log и завершает программу (аналог log.Fatalf).
func fatalf(log *logger.Logger, format string, args ...interface{}) {
	log.Error(fmt.Sprintf(format, args...))
	os.Exit(1)
}

func main() {
	flag.Parse()

	// 1. Проверка версии
	if showVersion || flag.Arg(0) == "version" {
		version.Print()
		return
	}

	// 2. Загрузка конфига
	cfg, err := config.Load[SchedulerConfig](configFile)
	if err != nil {
		fatalf(logger.New("info"), "Failed to load config: %v", err)
	}

	// 3. Применение секретов из ENV
	applySecrets(cfg)

	// 4. Инициализация кастомного логгера
	log := logger.New("info")
	log.Info(fmt.Sprintf("Starting Calendar Scheduler (version: %s)", version.String()))

	// 5. Graceful shutdown контекст
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// 6. Подключение к БД
	dsn := cfg.DB.DSN()
	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		fatalf(log, "failed to connect to PostgreSQL: %v", err)
	}

	// Настройки пула соединений
	cfg.DB.ApplyPoolSettings(db)

	storage := sqlstorage.New(db)
	defer storage.Close()
	log.Info("Connected to PostgreSQL")

	// 7. Подключение к RabbitMQ
	b := broker.NewAMQPBroker(cfg.RMQURL)
	if err := b.Connect(ctx); err != nil {
		fatalf(log, "failed to connect to RabbitMQ: %v", err)
	}
	defer b.Close()

	// 8. Создание структур в RabbitMQ
	if err := b.DeclareQueue(ctx, cfg.Queue); err != nil {
		fatalf(log, "failed to declare queue: %v", err)
	}
	if cfg.Exchange != "" {
		if err := b.DeclareExchange(ctx, cfg.Exchange); err != nil {
			fatalf(log, "failed to declare exchange: %v", err)
		}
		if err := b.BindQueue(ctx, cfg.Queue, cfg.Exchange, cfg.RoutingKey); err != nil {
			fatalf(log, "failed to bind queue to exchange: %v", err)
		}
	}
	log.Info("Connected to RabbitMQ")

	// 9. Запуск фоновых задач
	go runNotificationLoop(ctx, log, storage, b, cfg)
	go runCleanupLoop(ctx, log, storage, cfg)

	// 10. Ожидание сигнала завершения
	<-ctx.Done()
	log.Info("Shutting down scheduler...")
}

// applySecrets переопределяет чувствительные настройки из ENV.
func applySecrets(cfg *SchedulerConfig) {
	if pass := os.Getenv("DB_PASSWORD"); pass != "" {
		cfg.DB.Password = pass
	}
	if url := os.Getenv("RMQ_URL"); url != "" {
		cfg.RMQURL = url
	}
}

// runNotificationLoop периодически сканирует БД и публикует уведомления.
func runNotificationLoop(ctx context.Context, log *logger.Logger, storage *sqlstorage.Storage, b broker.Broker, cfg *SchedulerConfig) {
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := processNotifications(ctx, log, storage, b, cfg); err != nil {
				log.Error("notification loop error: " + err.Error())
			}
		}
	}
}

// processNotifications обрабатывает пакет уведомлений.
func processNotifications(ctx context.Context, log *logger.Logger, storage *sqlstorage.Storage, b broker.Broker, cfg *SchedulerConfig) error {
	notifications, err := storage.FetchPendingNotifications(ctx, 100)
	if err != nil {
		return fmt.Errorf("fetch notifications: %w", err)
	}
	if len(notifications) == 0 {
		return nil
	}

	var sentIDs []string
	for _, n := range notifications {
		body, err := json.Marshal(n)
		if err != nil {
			log.Error("marshal notification: " + err.Error())
			continue
		}

		if err := b.Publish(ctx, cfg.Exchange, cfg.RoutingKey, body); err != nil {
			log.Error("publish notification: " + err.Error())
			continue
		}
		sentIDs = append(sentIDs, n.EventID)
		log.Info("Notification published: " + n.Title + " (id=" + n.EventID + ")")
	}

	if len(sentIDs) > 0 {
		if err := storage.MarkNotificationsSent(ctx, sentIDs); err != nil {
			return fmt.Errorf("mark notifications sent: %w", err)
		}
	}
	return nil
}

// runCleanupLoop периодически удаляет старые события.
func runCleanupLoop(ctx context.Context, log *logger.Logger, storage *sqlstorage.Storage, cfg *SchedulerConfig) {
	ticker := time.NewTicker(cfg.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := cleanupOldEvents(ctx, log, storage); err != nil {
				log.Error("cleanup loop error: " + err.Error())
			}
		}
	}
}

// cleanupOldEvents удаляет старые завершившиеся события.
func cleanupOldEvents(ctx context.Context, log *logger.Logger, storage *sqlstorage.Storage) error {
	deleted, err := storage.DeleteOldEvents(ctx, 365*24*time.Hour)
	if err != nil {
		return fmt.Errorf("delete old events: %w", err)
	}
	if deleted > 0 {
		log.Info(fmt.Sprintf("Old events cleaned: %d deleted", deleted))
	}
	return nil
}
