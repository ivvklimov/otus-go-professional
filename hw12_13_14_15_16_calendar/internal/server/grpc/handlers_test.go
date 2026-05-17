package internalgrpc

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/api/proto"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/logger"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage"
)

// ==================== MOCKS ====================

// mockApp реализует интерфейс приложения для изоляции хэндлеров от бизнес-логики.
type mockApp struct {
	mock.Mock
}

// CreateEvent - новая сигнатура: без id в аргументах, возвращает (*Event, error).
func (m *mockApp) CreateEvent(ctx context.Context, title string, ownerID int64, description *string, dateStart, dateEnd time.Time, notifyAt *time.Time) (*storage.Event, error) {
	args := m.Called(ctx, title, ownerID, description, dateStart, dateEnd, notifyAt)
	return args.Get(0).(*storage.Event), args.Error(1)
}

func (m *mockApp) UpdateEvent(ctx context.Context, id string, event storage.Event) error {
	args := m.Called(ctx, id, event)
	return args.Error(0)
}

func (m *mockApp) DeleteEvent(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockApp) GetEvent(ctx context.Context, id string) (storage.Event, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(storage.Event), args.Error(1)
}

func (m *mockApp) ListEvents(ctx context.Context, ownerID int64, from, to time.Time) ([]storage.Event, error) {
	args := m.Called(ctx, ownerID, from, to)
	return args.Get(0).([]storage.Event), args.Error(1)
}

// ==================== TESTS ====================

func TestServer_CreateEvent(t *testing.T) {
	start := timestamppb.New(time.Date(2026, 5, 10, 10, 0, 0, 0, time.UTC))
	end := timestamppb.New(time.Date(2026, 5, 10, 11, 0, 0, 0, time.UTC))

	tests := []struct {
		name      string
		req       *proto.CreateEventRequest
		mockSetup func(m *mockApp)
		wantCode  codes.Code
		wantTitle string
	}{
		{
			name: "success",
			req: &proto.CreateEventRequest{
				Title:     "Meeting",
				DateStart: start,
				DateEnd:   end,
				OwnerId:   100,
			},
			mockSetup: func(m *mockApp) {
				// Новая сигнатура: без id, возвращаем (*Event, nil)
				m.On("CreateEvent", mock.Anything, "Meeting", int64(100),
					(*string)(nil), mock.Anything, mock.Anything, (*time.Time)(nil)).
					Return(&storage.Event{ID: "test-id", Title: "Meeting", OwnerID: 100}, nil)
			},
			wantCode:  codes.OK,
			wantTitle: "Meeting",
		},
		{
			name: "missing title",
			req: &proto.CreateEventRequest{
				DateStart: start,
				DateEnd:   end,
				OwnerId:   100,
			},
			mockSetup: func(_ *mockApp) {},
			wantCode:  codes.InvalidArgument,
		},
		{
			name: "missing dates",
			req: &proto.CreateEventRequest{
				Title:   "NoDates",
				OwnerId: 100,
			},
			mockSetup: func(_ *mockApp) {},
			wantCode:  codes.InvalidArgument,
		},
		{
			name: "date busy",
			req: &proto.CreateEventRequest{
				Title:     "Busy",
				DateStart: start,
				DateEnd:   end,
				OwnerId:   100,
			},
			mockSetup: func(m *mockApp) {
				m.On("CreateEvent", mock.Anything, "Busy", int64(100),
					(*string)(nil), mock.Anything, mock.Anything, (*time.Time)(nil)).
					Return((*storage.Event)(nil), storage.ErrDateBusy)
			},
			wantCode: codes.AlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockApp)
			tt.mockSetup(m)

			s := &Server{app: m, logger: logger.NewWithOutput("info", io.Discard)}
			resp, err := s.CreateEvent(context.Background(), tt.req)

			st, _ := status.FromError(err)
			assert.Equal(t, tt.wantCode, st.Code(), "unexpected gRPC status code")

			if tt.wantCode == codes.OK {
				assert.NotNil(t, resp)
				assert.Equal(t, tt.wantTitle, resp.Event.Title)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestServer_GetEvent(t *testing.T) {
	tests := []struct {
		name      string
		req       *proto.GetEventRequest
		mockSetup func(m *mockApp)
		wantCode  codes.Code
		wantID    string
	}{
		{
			name: "success",
			req:  &proto.GetEventRequest{Id: "evt_123"},
			mockSetup: func(m *mockApp) {
				m.On("GetEvent", mock.Anything, "evt_123").
					Return(storage.Event{ID: "evt_123", Title: "Found"}, nil)
			},
			wantCode: codes.OK,
			wantID:   "evt_123",
		},
		{
			name:      "missing id",
			req:       &proto.GetEventRequest{Id: ""},
			mockSetup: func(_ *mockApp) {},
			wantCode:  codes.InvalidArgument,
		},
		{
			name: "not found",
			req:  &proto.GetEventRequest{Id: "missing"},
			mockSetup: func(m *mockApp) {
				m.On("GetEvent", mock.Anything, "missing").
					Return(storage.Event{}, storage.ErrNotFound)
			},
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockApp)
			tt.mockSetup(m)

			s := &Server{app: m, logger: logger.NewWithOutput("info", io.Discard)}
			resp, err := s.GetEvent(context.Background(), tt.req)

			st, _ := status.FromError(err)
			assert.Equal(t, tt.wantCode, st.Code())

			if tt.wantCode == codes.OK {
				assert.Equal(t, tt.wantID, resp.Event.Id)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestServer_UpdateEvent(t *testing.T) {
	tests := []struct {
		name      string
		req       *proto.UpdateEventRequest
		mockSetup func(m *mockApp)
		wantCode  codes.Code
	}{
		{
			name: "success partial update",
			req:  &proto.UpdateEventRequest{Id: "evt_1", Title: "NewTitle"},
			mockSetup: func(m *mockApp) {
				m.On("UpdateEvent", mock.Anything, "evt_1", mock.AnythingOfType("storage.Event")).Return(nil)
				m.On("GetEvent", mock.Anything, "evt_1").
					Return(storage.Event{ID: "evt_1", Title: "NewTitle"}, nil)
			},
			wantCode: codes.OK,
		},
		{
			name:      "missing id",
			req:       &proto.UpdateEventRequest{Title: "NoID"},
			mockSetup: func(_ *mockApp) {},
			wantCode:  codes.InvalidArgument,
		},
		{
			name: "not found",
			req:  &proto.UpdateEventRequest{Id: "missing"},
			mockSetup: func(m *mockApp) {
				m.On("UpdateEvent", mock.Anything, "missing", mock.Anything).
					Return(storage.ErrNotFound)
			},
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockApp)
			tt.mockSetup(m)

			s := &Server{app: m, logger: logger.NewWithOutput("info", io.Discard)}
			_, err := s.UpdateEvent(context.Background(), tt.req)

			st, _ := status.FromError(err)
			assert.Equal(t, tt.wantCode, st.Code())
			m.AssertExpectations(t)
		})
	}
}

func TestServer_DeleteEvent(t *testing.T) {
	tests := []struct {
		name      string
		req       *proto.DeleteEventRequest
		mockSetup func(m *mockApp)
		wantCode  codes.Code
	}{
		{
			name: "success",
			req:  &proto.DeleteEventRequest{Id: "evt_1"},
			mockSetup: func(m *mockApp) {
				m.On("DeleteEvent", mock.Anything, "evt_1").Return(nil)
			},
			wantCode: codes.OK,
		},
		{
			name:      "missing id",
			req:       &proto.DeleteEventRequest{Id: ""},
			mockSetup: func(_ *mockApp) {},
			wantCode:  codes.InvalidArgument,
		},
		{
			name: "not found",
			req:  &proto.DeleteEventRequest{Id: "missing"},
			mockSetup: func(m *mockApp) {
				m.On("DeleteEvent", mock.Anything, "missing").Return(storage.ErrNotFound)
			},
			wantCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockApp)
			tt.mockSetup(m)

			s := &Server{app: m, logger: logger.NewWithOutput("info", io.Discard)}
			resp, err := s.DeleteEvent(context.Background(), tt.req)

			st, _ := status.FromError(err)
			assert.Equal(t, tt.wantCode, st.Code())

			if tt.wantCode == codes.OK {
				assert.True(t, resp.Deleted)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestServer_ListEventsForDay(t *testing.T) {
	m := new(mockApp)
	m.On("ListEvents", mock.Anything, int64(55),
		mock.MatchedBy(func(t time.Time) bool { return t.Hour() == 0 && t.Minute() == 0 }),
		mock.MatchedBy(func(t time.Time) bool { return t.Hour() == 23 && t.Minute() == 59 }),
	).Return([]storage.Event{{ID: "e1", Title: "DayEvt"}}, nil)

	s := &Server{app: m, logger: logger.NewWithOutput("info", io.Discard)}
	req := &proto.ListEventsForDayRequest{
		OwnerId: 55,
		Date:    timestamppb.New(time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)),
	}

	resp, err := s.ListEventsForDay(context.Background(), req)

	assert.NoError(t, err)
	assert.Len(t, resp.Events, 1)
	assert.Equal(t, "DayEvt", resp.Events[0].Title)
	m.AssertExpectations(t)
}

func TestServer_HealthCheck(t *testing.T) {
	s := &Server{logger: logger.NewWithOutput("info", io.Discard)}
	resp, err := s.HealthCheck(context.Background(), &emptypb.Empty{})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
}
