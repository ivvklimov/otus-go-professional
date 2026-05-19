package internalgrpc

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/api/proto"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/app"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/logger"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage"
)

// Server - gRPC-сервер календаря.
type Server struct {
	proto.UnimplementedCalendarServiceServer

	app    app.Service // интерфейс вместо конкретной структуры
	logger *logger.Logger
}

// New создаёт экземпляр сервера.
func New(app *app.App, logger *logger.Logger) *Server {
	return &Server{
		app:    app,
		logger: logger,
	}
}

// Register регистрирует сервис в gRPC-сервере.
func (s *Server) Register(grpcServer *grpc.Server) {
	proto.RegisterCalendarServiceServer(grpcServer, s)
}

// ==================== Helpers: маппинг ====================

// toProtoEvent конвертирует storage.Event -> proto.Event.
func toProtoEvent(e *storage.Event) *proto.Event {
	if e == nil {
		return nil
	}
	return &proto.Event{
		Id:          e.ID,
		Title:       e.Title,
		Description: strPtrToStr(e.Description),
		OwnerId:     e.OwnerID,
		DateStart:   timestamppb.New(e.DateStart),
		DateEnd:     timestamppb.New(e.DateEnd),
		NotifyAt:    timePtrToTimestamp(e.NotifyAt),
		CreatedAt:   timestamppb.New(e.CreatedAt),
		UpdatedAt:   timestamppb.New(e.UpdatedAt),
	}
}

// ===== Helpers для опциональных полей =====

// strPtrToStr: *string -> string (пустая строка если nil).
func strPtrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// timePtrToTimestamp: *time.Time -> *timestamppb.Timestamp.
func timePtrToTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
