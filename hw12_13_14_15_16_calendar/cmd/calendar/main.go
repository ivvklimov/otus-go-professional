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
)

var (
	configFile  string
	showVersion bool
)

func init() {
	flag.StringVar(&configFile, "config", "configs/config.yaml", "Path to configuration file")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
}

func main() {
	flag.Parse()

	// Проверка версии: -version или version
	if showVersion || flag.Arg(0) == "version" {
		printVersion()
		return
	}

	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	logg := logger.New(cfg.Logger.Level)

	storageImpl, closer, err := createStorage(cfg, logg)
	if err != nil {
		logg.Error("failed to create storage: " + err.Error())
		os.Exit(1)
	}
	if closer != nil {
		defer closer.Close()
	}

	calendar := app.New(logg, storageImpl)

	server := internalhttp.NewServer(
		cfg.Server.Host,
		cfg.Server.Port,
		logg,
		calendar,
	)

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	// ==================== gRPC Server ====================
	var grpcServer *grpc.Server

	grpcAddr := ":" + cfg.GRPC.Port
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		logg.Error("failed to listen gRPC: " + err.Error())
		os.Exit(1)
	}

	grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(internalgrpc.LoggingInterceptor(logg)),
	)
	internalgrpc.New(calendar, logg).Register(grpcServer)
	reflection.Register(grpcServer) // для дебага, чтобы не указывать путь дло proto файлов

	go func() {
		logg.Info("gRPC server starting at " + grpcAddr)
		if err := grpcServer.Serve(grpcListener); err != nil {
			logg.Error("gRPC server error: " + err.Error())
		}
	}()
	// ==================== /gRPC Server ====================

	go func() {
		<-ctx.Done()

		// Остановка gRPC
		if grpcServer != nil {
			grpcServer.GracefulStop()
			logg.Info("gRPC server stopped")
		}

		// Остановка HTTP
		ctxShutdown, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		if err := server.Stop(ctxShutdown); err != nil {
			logg.Error("failed to stop http server: " + err.Error())
		}
	}()

	logg.Info("calendar is running... (storage: " + cfg.Storage.Type + ")")

	if err := server.Start(ctx); err != nil {
		logg.Error("failed to start http server: " + err.Error())
		cancel()
		os.Exit(1)
	}
}

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
