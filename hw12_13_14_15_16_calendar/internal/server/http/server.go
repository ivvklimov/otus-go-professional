package internalhttp

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/app"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/logger"
)

// Server - HTTP-сервер с поддержкой graceful shutdown.
type Server struct {
	httpServer *http.Server
	logger     *logger.Logger
}

// NewServer создаёт новый сервер.
func NewServer(host string, port int, logger *logger.Logger, app app.Service) *Server {
	mux := http.NewServeMux()

	// Создаём хэндлер с внедрёнными зависимостями
	h := NewHandler(app, logger)

	// API v1 routes
	mux.HandleFunc("POST /api/v1/calendar/events", h.CreateEvent)
	mux.HandleFunc("GET /api/v1/calendar/events/", h.GetEvent)
	mux.HandleFunc("PUT /api/v1/calendar/events/", h.UpdateEvent)
	mux.HandleFunc("DELETE /api/v1/calendar/events/", h.DeleteEvent)
	mux.HandleFunc("GET /api/v1/calendar/events/day", h.ListEventsForDay)
	mux.HandleFunc("GET /api/v1/calendar/events/week", h.ListEventsForWeek)
	mux.HandleFunc("GET /api/v1/calendar/events/month", h.ListEventsForMonth)

	// Применяем middleware логирования
	handler := LoggingMiddleware(logger, mux)

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
