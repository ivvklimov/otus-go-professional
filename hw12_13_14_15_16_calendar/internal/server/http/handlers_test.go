package internalhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/logger"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage"
)

// ==================== MOCKS ====================

// mockApp реализует интерфейс app.Service для изоляции хэндлеров от бизнес-логики.
type mockApp struct {
	mock.Mock
}

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

// ==================== HELPERS ====================

// newTestRequest создаёт *http.Request с нужным методом, путём и телом.
func newTestRequest(method, path string, body interface{}) *http.Request {
	var r io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		r = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// readResponseBody читает и парсит тело ответа.
func readResponseBody(w *httptest.ResponseRecorder, v interface{}) error {
	return json.NewDecoder(w.Body).Decode(v)
}

// ==================== TESTS ====================

func TestHandler_CreateEvent(t *testing.T) {
	tests := []struct {
		name      string
		reqBody   CreateEventRequest
		mockSetup func(m *mockApp)
		wantCode  int
		wantJSON  string // подстрока, которую ожидаем в ответе
	}{
		{
			name: "success",
			reqBody: CreateEventRequest{
				Title:     "Meeting",
				DateStart: "2026-05-10T10:00:00Z",
				DateEnd:   "2026-05-10T11:00:00Z",
				OwnerID:   100,
			},
			mockSetup: func(m *mockApp) {
				m.On("CreateEvent", mock.Anything, "Meeting", int64(100),
					(*string)(nil), mock.Anything, mock.Anything, (*time.Time)(nil)).
					Return(&storage.Event{ID: "test-id", Title: "Meeting", OwnerID: 100}, nil)
			},
			wantCode: http.StatusCreated,
			wantJSON: `"title":"Meeting"`,
		},
		{
			name: "missing title",
			reqBody: CreateEventRequest{
				DateStart: "2026-05-10T10:00:00Z",
				DateEnd:   "2026-05-10T11:00:00Z",
				OwnerID:   100,
			},
			mockSetup: func(_ *mockApp) {},
			wantCode:  http.StatusBadRequest,
		},
		{
			name: "missing dates",
			reqBody: CreateEventRequest{
				Title:   "NoDates",
				OwnerID: 100,
			},
			mockSetup: func(_ *mockApp) {},
			wantCode:  http.StatusBadRequest,
		},
		{
			name: "invalid date format",
			reqBody: CreateEventRequest{
				Title:     "BadDate",
				DateStart: "not-a-date",
				DateEnd:   "2026-05-10T11:00:00Z",
				OwnerID:   100,
			},
			mockSetup: func(_ *mockApp) {},
			wantCode:  http.StatusBadRequest,
		},
		{
			name: "date busy",
			reqBody: CreateEventRequest{
				Title:     "Busy",
				DateStart: "2026-05-10T10:00:00Z",
				DateEnd:   "2026-05-10T11:00:00Z",
				OwnerID:   100,
			},
			mockSetup: func(m *mockApp) {
				m.On("CreateEvent", mock.Anything, "Busy", int64(100),
					(*string)(nil), mock.Anything, mock.Anything, (*time.Time)(nil)).
					Return((*storage.Event)(nil), storage.ErrDateBusy)
			},
			wantCode: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockApp)
			tt.mockSetup(m)

			h := NewHandler(m, logger.NewWithOutput("info", io.Discard))

			req := newTestRequest(http.MethodPost, "/api/v1/calendar/events", tt.reqBody)
			w := httptest.NewRecorder()

			h.CreateEvent(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
			if tt.wantJSON != "" {
				assert.Contains(t, w.Body.String(), tt.wantJSON)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestHandler_GetEvent(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		mockSetup func(m *mockApp)
		wantCode  int
		wantID    string
	}{
		{
			name: "success",
			path: "/api/v1/calendar/events/evt_123",
			mockSetup: func(m *mockApp) {
				m.On("GetEvent", mock.Anything, "evt_123").
					Return(storage.Event{ID: "evt_123", Title: "Found"}, nil)
			},
			wantCode: http.StatusOK,
			wantID:   "evt_123",
		},
		{
			name:      "missing id",
			path:      "/api/v1/calendar/events/",
			mockSetup: func(_ *mockApp) {},
			wantCode:  http.StatusBadRequest,
		},
		{
			name: "not found",
			path: "/api/v1/calendar/events/missing",
			mockSetup: func(m *mockApp) {
				m.On("GetEvent", mock.Anything, "missing").
					Return(storage.Event{}, storage.ErrNotFound)
			},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockApp)
			tt.mockSetup(m)

			h := NewHandler(m, logger.NewWithOutput("info", io.Discard))

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			h.GetEvent(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
			if tt.wantCode == http.StatusOK {
				var resp EventResponse
				_ = readResponseBody(w, &resp)
				assert.Equal(t, tt.wantID, resp.Event.ID)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestHandler_UpdateEvent(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		reqBody   UpdateEventRequest
		mockSetup func(m *mockApp)
		wantCode  int
	}{
		{
			name: "success partial update",
			path: "/api/v1/calendar/events/evt_1",
			reqBody: UpdateEventRequest{
				Title: stringPtr("NewTitle"),
			},
			mockSetup: func(m *mockApp) {
				m.On("UpdateEvent", mock.Anything, "evt_1", mock.AnythingOfType("storage.Event")).Return(nil)
				m.On("GetEvent", mock.Anything, "evt_1").
					Return(storage.Event{ID: "evt_1", Title: "NewTitle"}, nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name:      "missing id",
			path:      "/api/v1/calendar/events/",
			reqBody:   UpdateEventRequest{Title: stringPtr("NoID")},
			mockSetup: func(_ *mockApp) {},
			wantCode:  http.StatusBadRequest,
		},
		{
			name: "not found",
			path: "/api/v1/calendar/events/missing",
			reqBody: UpdateEventRequest{
				Title: stringPtr("Updated"),
			},
			mockSetup: func(m *mockApp) {
				m.On("UpdateEvent", mock.Anything, "missing", mock.Anything).
					Return(storage.ErrNotFound)
			},
			wantCode: http.StatusNotFound,
		},
		{
			name: "date busy",
			path: "/api/v1/calendar/events/evt_1",
			reqBody: UpdateEventRequest{
				DateStart: stringPtr("2026-05-10T10:30:00Z"),
			},
			mockSetup: func(m *mockApp) {
				m.On("UpdateEvent", mock.Anything, "evt_1", mock.Anything).
					Return(storage.ErrDateBusy)
			},
			wantCode: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockApp)
			tt.mockSetup(m)

			h := NewHandler(m, logger.NewWithOutput("info", io.Discard))

			req := newTestRequest(http.MethodPut, tt.path, tt.reqBody)
			w := httptest.NewRecorder()

			h.UpdateEvent(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
			m.AssertExpectations(t)
		})
	}
}

func TestHandler_DeleteEvent(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		mockSetup func(m *mockApp)
		wantCode  int
	}{
		{
			name: "success",
			path: "/api/v1/calendar/events/evt_1",
			mockSetup: func(m *mockApp) {
				m.On("DeleteEvent", mock.Anything, "evt_1").Return(nil)
			},
			wantCode: http.StatusOK,
		},
		{
			name:      "missing id",
			path:      "/api/v1/calendar/events/",
			mockSetup: func(_ *mockApp) {},
			wantCode:  http.StatusBadRequest,
		},
		{
			name: "not found",
			path: "/api/v1/calendar/events/missing",
			mockSetup: func(m *mockApp) {
				m.On("DeleteEvent", mock.Anything, "missing").Return(storage.ErrNotFound)
			},
			wantCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockApp)
			tt.mockSetup(m)

			h := NewHandler(m, logger.NewWithOutput("info", io.Discard))

			req := httptest.NewRequest(http.MethodDelete, tt.path, nil)
			w := httptest.NewRecorder()

			h.DeleteEvent(w, req)

			assert.Equal(t, tt.wantCode, w.Code)
			if tt.wantCode == http.StatusOK {
				var resp DeleteResponse
				_ = readResponseBody(w, &resp)
				assert.True(t, resp.Deleted)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestHandler_ListEventsForDay(t *testing.T) {
	m := new(mockApp)
	m.On("ListEvents", mock.Anything, int64(55),
		mock.MatchedBy(func(t time.Time) bool { return t.Hour() == 0 }),
		mock.MatchedBy(func(t time.Time) bool { return t.Hour() == 23 }),
	).Return([]storage.Event{{ID: "e1", Title: "DayEvt"}}, nil)

	h := NewHandler(m, logger.NewWithOutput("info", io.Discard))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/calendar/events/day?owner_id=55&date=2026-06-15T00:00:00Z", nil)
	w := httptest.NewRecorder()

	h.ListEventsForDay(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp ListEventsResponse
	_ = readResponseBody(w, &resp)
	assert.Len(t, resp.Events, 1)
	assert.Equal(t, "DayEvt", resp.Events[0].Title)
	m.AssertExpectations(t)
}

// stringPtr - хелпер для получения *string из string.
func stringPtr(s string) *string {
	return &s
}
