package internalhttp

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_calendar/internal/logger"
)

// Server — HTTP-сервер с поддержкой graceful shutdown.
type Server struct {
	httpServer *http.Server
	logger     *logger.Logger
}

// NewServer создаёт новый сервер.
func NewServer(host string, port int, logger *logger.Logger, app interface{}) *Server {
	mux := http.NewServeMux()

	// ДЗ №12: только один эндпоинт /hello
	mux.HandleFunc("/hello", helloHandler)

	// Применяем middleware логирования
	handler := LoggingMiddleware(logger, mux)

	_ = app

	return &Server{
		httpServer: &http.Server{
			Addr:         host + ":" + strconv.Itoa(port),
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

// Start запускает сервер в блокирующем режиме.
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("starting http server addr=" + s.httpServer.Addr)

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return s.Stop(context.Background())
	}
}

// Stop выполняет graceful shutdown.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("stopping http server")
	return s.httpServer.Shutdown(ctx)
}
