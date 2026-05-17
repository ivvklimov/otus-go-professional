package internalgrpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/api/proto"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage"
)

// CreateEvent создаёт новое событие.
func (s *Server) CreateEvent(ctx context.Context, req *proto.CreateEventRequest) (*proto.CreateEventResponse, error) {
	s.logger.Info("gRPC CreateEvent: title=" + req.Title)

	// Валидация обязательных полей
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.DateStart == nil || req.DateEnd == nil {
		return nil, status.Error(codes.InvalidArgument, "dateStart and dateEnd are required")
	}
	if req.OwnerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "ownerId is required")
	}

	dateStart := req.DateStart.AsTime()
	dateEnd := req.DateEnd.AsTime()
	if dateEnd.Before(dateStart) {
		return nil, status.Error(codes.InvalidArgument, "dateEnd must be after dateStart")
	}

	// Опциональные поля
	var description *string
	if req.Description != "" {
		description = &req.Description
	}
	var notifyAt *time.Time
	if req.NotifyAt != nil {
		t := req.NotifyAt.AsTime()
		notifyAt = &t
	}

	// Вызываем app - ID сгенерирует хранилище, результат вернётся в evt
	evt, err := s.app.CreateEvent(ctx, req.Title, req.OwnerId, description, dateStart, dateEnd, notifyAt)
	if err != nil {
		if errors.Is(err, storage.ErrDateBusy) {
			return nil, status.Error(codes.AlreadyExists, "date busy")
		}
		s.logger.Error("gRPC CreateEvent failed: " + err.Error())
		return nil, status.Error(codes.Internal, "failed to create event")
	}

	// Возвращаем созданное событие (с заполненным ID из хранилища)
	return &proto.CreateEventResponse{
		Event: toProtoEvent(evt),
	}, nil
}

// UpdateEvent обновляет существующее событие.
func (s *Server) UpdateEvent(ctx context.Context, req *proto.UpdateEventRequest) (*proto.UpdateEventResponse, error) {
	s.logger.Info("gRPC UpdateEvent: id=" + req.Id)

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	// app.UpdateEvent принимает storage.Event по значению
	evt := storage.Event{
		ID:      req.Id,
		Title:   req.Title,
		OwnerID: req.OwnerId,
	}
	if req.Description != "" {
		evt.Description = &req.Description
	}
	if req.DateStart != nil {
		evt.DateStart = req.DateStart.AsTime()
	}
	if req.DateEnd != nil {
		evt.DateEnd = req.DateEnd.AsTime()
	}
	if req.NotifyAt != nil {
		t := req.NotifyAt.AsTime()
		evt.NotifyAt = &t
	}

	err := s.app.UpdateEvent(ctx, req.Id, evt)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "event not found")
		}
		s.logger.Error("gRPC UpdateEvent failed: " + err.Error())
		return nil, status.Error(codes.Internal, "failed to update event")
	}

	// Получаем актуальную версию из БД, чтобы вернуть полный объект
	updated, err := s.app.GetEvent(ctx, req.Id)
	if err != nil {
		s.logger.Error("gRPC UpdateEvent: failed to fetch updated event: " + err.Error())
		return nil, status.Error(codes.Internal, "failed to fetch updated event")
	}

	return &proto.UpdateEventResponse{
		Event: toProtoEvent(&updated),
	}, nil
}

// DeleteEvent удаляет событие.
func (s *Server) DeleteEvent(ctx context.Context, req *proto.DeleteEventRequest) (*proto.DeleteEventResponse, error) {
	s.logger.Info("gRPC DeleteEvent: id=" + req.Id)

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	err := s.app.DeleteEvent(ctx, req.Id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "event not found")
		}
		s.logger.Error("gRPC DeleteEvent failed: " + err.Error())
		return nil, status.Error(codes.Internal, "failed to delete event")
	}

	return &proto.DeleteEventResponse{Deleted: true}, nil
}

// GetEvent возвращает событие по ID.
func (s *Server) GetEvent(ctx context.Context, req *proto.GetEventRequest) (*proto.GetEventResponse, error) {
	s.logger.Info("gRPC GetEvent: id=" + req.Id)

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	event, err := s.app.GetEvent(ctx, req.Id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "event not found")
		}
		s.logger.Error("gRPC GetEvent failed: " + err.Error())
		return nil, status.Error(codes.Internal, "failed to get event")
	}

	return &proto.GetEventResponse{Event: toProtoEvent(&event)}, nil
}

// ListEventsForDay возвращает события владельца за указанный день.
func (s *Server) ListEventsForDay(ctx context.Context, req *proto.ListEventsForDayRequest) (*proto.ListEventsResponse, error) {
	s.logger.Info(fmt.Sprintf("gRPC ListEventsForDay: owner_id=%d date=%s", req.OwnerId, req.Date.AsTime().Format("2006-01-02")))

	if req.OwnerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "owner_id is required")
	}
	if req.Date == nil {
		return nil, status.Error(codes.InvalidArgument, "date is required")
	}

	// Рассчитываем диапазон: весь день с 00:00:00 до 23:59:59.999
	date := req.Date.AsTime()
	from := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	to := from.Add(24*time.Hour - time.Nanosecond)

	events, err := s.app.ListEvents(ctx, req.OwnerId, from, to)
	if err != nil {
		s.logger.Error("gRPC ListEventsForDay failed: " + err.Error())
		return nil, status.Error(codes.Internal, "failed to list events for day")
	}

	return eventsToResponse(events), nil
}

// ListEventsForWeek возвращает события владельца за неделю, в которой находится date.
// Неделя считается с понедельника по воскресенье.
func (s *Server) ListEventsForWeek(ctx context.Context, req *proto.ListEventsForWeekRequest) (*proto.ListEventsResponse, error) {
	s.logger.Info(fmt.Sprintf("gRPC ListEventsForWeek: owner_id=%d date=%s", req.OwnerId, req.Date.AsTime().Format("2006-01-02")))

	if req.OwnerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "owner_id is required")
	}
	if req.Date == nil {
		return nil, status.Error(codes.InvalidArgument, "date is required")
	}

	// Рассчитываем диапазон: понедельник 00:00:00 - воскресенье 23:59:59.999
	date := req.Date.AsTime()
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday - 7 для удобства расчёта
	}
	monday := date.AddDate(0, 0, -(weekday - 1))
	from := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, date.Location())
	to := from.Add(7*24*time.Hour - time.Nanosecond)

	events, err := s.app.ListEvents(ctx, req.OwnerId, from, to)
	if err != nil {
		s.logger.Error("gRPC ListEventsForWeek failed: " + err.Error())
		return nil, status.Error(codes.Internal, "failed to list events for week")
	}

	return eventsToResponse(events), nil
}

// ListEventsForMonth возвращает события владельца за месяц, в котором находится date.
func (s *Server) ListEventsForMonth(ctx context.Context, req *proto.ListEventsForMonthRequest) (*proto.ListEventsResponse, error) {
	s.logger.Info(fmt.Sprintf("gRPC ListEventsForMonth: owner_id=%d date=%s", req.OwnerId, req.Date.AsTime().Format("2006-01-02")))

	if req.OwnerId == 0 {
		return nil, status.Error(codes.InvalidArgument, "owner_id is required")
	}
	if req.Date == nil {
		return nil, status.Error(codes.InvalidArgument, "date is required")
	}

	// Рассчитываем диапазон: 1-е число 00:00:00 - последний день 23:59:59.999
	date := req.Date.AsTime()
	from := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	to := from.AddDate(0, 1, 0).Add(-time.Nanosecond)

	events, err := s.app.ListEvents(ctx, req.OwnerId, from, to)
	if err != nil {
		s.logger.Error("gRPC ListEventsForMonth failed: " + err.Error())
		return nil, status.Error(codes.Internal, "failed to list events for month")
	}

	return eventsToResponse(events), nil
}

// eventsToResponse - хелпер для конвертации []storage.Event -> *proto.ListEventsResponse.
func eventsToResponse(events []storage.Event) *proto.ListEventsResponse {
	protoEvents := make([]*proto.Event, 0, len(events))
	for _, e := range events {
		evt := e // копируем, чтобы взять адрес
		protoEvents = append(protoEvents, toProtoEvent(&evt))
	}
	return &proto.ListEventsResponse{
		Events: protoEvents,
		Total:  int32(len(protoEvents)),
	}
}

// HealthCheck проверяет работоспособность сервиса.
func (s *Server) HealthCheck(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	s.logger.Info("gRPC HealthCheck")
	return &emptypb.Empty{}, nil
}
