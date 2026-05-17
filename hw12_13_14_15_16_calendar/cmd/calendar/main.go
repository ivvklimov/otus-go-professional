package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/app"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/config"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/logger"
	internalgrpc "github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/server/grpc"
	internalhttp "github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/server/http"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage"
	memorystorage "github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage/memory"
	sqlstorage "github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage/sql"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/version"
)

// Глобальные переменные для флагов.
var (
	configFile  string
	showVersion bool
)

// init регистрирует флаги (выполняется до main).
func init() {
	flag.StringVar(&configFile, "config", "configs/calendar.yaml", "Path to configuration file")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
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
	cfg, err := config.Load[config.Config](configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 3. Применяем ENV-переменные (приоритет над файлом)
	// Это позволяет переопределять настройки через Docker/K8s без пересборки
	config.ApplyAPIEnvOverrides(cfg)

	// 4. Валидация конфига (fail-fast)
	// Лучше упасть сразу с понятной ошибкой, чем работать с мусором
	if err := config.ValidateAPI(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Config validation failed: %v\n", err)
		os.Exit(1)
	}

	// 5. Инициализация логгера (теперь конфиг гарантированно валиден)
	log := logger.New(cfg.Logger.Level)
	log.Info(fmt.Sprintf("Starting Calendar API (version: %s)", version.String()))

	// 6. Инициализация хранилища
	storageImpl, closer, err := createStorage(cfg, log)
	if err != nil {
		log.Error("failed to create storage: " + err.Error())
		os.Exit(1)
	}
	if closer != nil {
		defer closer.Close()
	}

	// 7. Инициализация приложения
	calendar := app.New(log, storageImpl)

	// 8. Запуск HTTP сервера
	server := internalhttp.NewServer(
		cfg.Server.Host,
		cfg.Server.Port,
		log,
		calendar,
	)

	// 9. Контекст с graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	// ==================== gRPC Server ====================
	var grpcServer *grpc.Server

	grpcAddr := ":" + cfg.GRPC.Port
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Error("failed to listen gRPC: " + err.Error())
		os.Exit(1)
	}

	grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(internalgrpc.LoggingInterceptor(log)),
	)
	internalgrpc.New(calendar, log).Register(grpcServer)
	reflection.Register(grpcServer) // для дебага

	go func() {
		log.Info("gRPC server starting at " + grpcAddr)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Error("gRPC server error: " + err.Error())
		}
	}()
	// ==================== /gRPC Server ====================

	// 10. Graceful shutdown handler
	go func() {
		<-ctx.Done()

		// Остановка gRPC
		if grpcServer != nil {
			grpcServer.GracefulStop()
			log.Info("gRPC server stopped")
		}

		// Остановка HTTP
		ctxShutdown, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		if err := server.Stop(ctxShutdown); err != nil {
			log.Error("failed to stop http server: " + err.Error())
		}
	}()

	log.Info("calendar is running... (storage: " + cfg.Storage.Type + ")")

	if err := server.Start(ctx); err != nil {
		log.Error("failed to start http server: " + err.Error())
		cancel()
		os.Exit(1)
	}
}

// createStorage создаёт хранилище на основе конфига.
func createStorage(cfg *config.Config, log *logger.Logger) (storage.Storage, interface{ Close() error }, error) {
	switch cfg.Storage.Type {
	case "memory", "":
		log.Info("using in-memory storage")
		return memorystorage.New(), nil, nil

	case "sql":
		// DSN логируем как простую строку
		log.Info("using SQL storage: " + cfg.DB.DSN())

		db, err := sqlx.Connect("postgres", cfg.DB.DSN())
		if err != nil {
			return nil, nil, fmt.Errorf("connect to postgres: %w", err)
		}
		db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
		if err := db.Ping(); err != nil {
			_ = db.Close()
			return nil, nil, fmt.Errorf("ping postgres: %w", err)
		}
		log.Info("connected to PostgreSQL")
		return sqlstorage.New(db), db, nil

	default:
		return nil, nil, fmt.Errorf("unknown storage type: %q", cfg.Storage.Type)
	}
}
