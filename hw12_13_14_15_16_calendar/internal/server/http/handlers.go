package internalhttp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/app"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/logger"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage"
)

// Handler обрабатывает HTTP-запросы к API календаря.
type Handler struct {
	app    app.Service
	logger *logger.Logger
}

// NewHandler создаёт новый обработчик.
func NewHandler(app app.Service, logger *logger.Logger) *Handler {
	return &Handler{app: app, logger: logger}
}

// ==================== CREATE EVENT ====================

// CreateEvent обрабатывает POST /api/v1/calendar/events.
func (h *Handler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация обязательных полей
	if req.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	if req.DateStart == "" || req.DateEnd == "" {
		http.Error(w, "dateStart and dateEnd are required", http.StatusBadRequest)
		return
	}
	if req.OwnerID == 0 {
		http.Error(w, "ownerId is required", http.StatusBadRequest)
		return
	}

	// Парсинг дат
	dateStart, err := parseRFC3339(req.DateStart)
	if err != nil || dateStart == nil {
		http.Error(w, "invalid dateStart format (use RFC3339)", http.StatusBadRequest)
		return
	}
	dateEnd, err := parseRFC3339(req.DateEnd)
	if err != nil || dateEnd == nil {
		http.Error(w, "invalid dateEnd format (use RFC3339)", http.StatusBadRequest)
		return
	}
	if dateEnd.Before(*dateStart) {
		http.Error(w, "dateEnd must be after dateStart", http.StatusBadRequest)
		return
	}

	// Опциональные поля
	var description *string
	if req.Description != "" {
		description = &req.Description
	}
	var notifyAt *time.Time
	if req.NotifyAt != "" {
		notifyAt, err = parseRFC3339(req.NotifyAt)
		if err != nil {
			http.Error(w, "invalid notifyAt format (use RFC3339)", http.StatusBadRequest)
			return
		}
	}

	// Вызываем app - ID сгенерирует хранилище, результат вернётся в evt
	evt, err := h.app.CreateEvent(r.Context(), req.Title, req.OwnerID, description, *dateStart, *dateEnd, notifyAt)
	if err != nil {
		if errors.Is(err, storage.ErrDateBusy) {
			http.Error(w, "date busy", http.StatusConflict)
			return
		}
		h.logger.Error("CreateEvent failed: " + err.Error())
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Возвращаем созданное событие (с заполненным ID из хранилища)
	// Не делаем лишний GetEvent - evt уже содержит все данные
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(EventResponse{Event: *evt})
}

// ==================== GET EVENT ====================

// GetEvent обрабатывает GET /api/v1/calendar/events/{id}.
func (h *Handler) GetEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID из пути: /api/v1/calendar/events/{id}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/calendar/events/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	evt, err := h.app.GetEvent(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "event not found", http.StatusNotFound)
			return
		}
		h.logger.Error("GetEvent failed: " + err.Error())
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(EventResponse{Event: evt})
}

// ==================== UPDATE EVENT ====================

// UpdateEvent обрабатывает PUT /api/v1/calendar/events/{id}.
func (h *Handler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/calendar/events/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	var req UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Собираем частичное обновление
	evt := storage.Event{ID: id}
	if req.Title != nil {
		if *req.Title == "" {
			http.Error(w, "title cannot be empty", http.StatusBadRequest)
			return
		}
		evt.Title = *req.Title
	}
	if req.Description != nil {
		evt.Description = req.Description
	}
	if req.DateStart != nil {
		ds, err := parseRFC3339(*req.DateStart)
		if err != nil || ds == nil {
			http.Error(w, "invalid dateStart format", http.StatusBadRequest)
			return
		}
		evt.DateStart = *ds
	}
	if req.DateEnd != nil {
		de, err := parseRFC3339(*req.DateEnd)
		if err != nil || de == nil {
			http.Error(w, "invalid dateEnd format", http.StatusBadRequest)
			return
		}
		evt.DateEnd = *de
	}
	if req.OwnerID != nil {
		evt.OwnerID = *req.OwnerID
	}
	if req.NotifyAt != nil {
		na, err := parseRFC3339(*req.NotifyAt)
		if err != nil {
			http.Error(w, "invalid notifyAt format", http.StatusBadRequest)
			return
		}
		evt.NotifyAt = na
	}

	err := h.app.UpdateEvent(r.Context(), id, evt)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "event not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, storage.ErrDateBusy) {
			http.Error(w, "date busy", http.StatusConflict)
			return
		}
		h.logger.Error("UpdateEvent failed: " + err.Error())
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Возвращаем обновлённое событие
	updated, _ := h.app.GetEvent(r.Context(), id)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(EventResponse{Event: updated})
}

// ==================== DELETE EVENT ====================

// DeleteEvent обрабатывает DELETE /api/v1/calendar/events/{id}.
func (h *Handler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/v1/calendar/events/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	err := h.app.DeleteEvent(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "event not found", http.StatusNotFound)
			return
		}
		h.logger.Error("DeleteEvent failed: " + err.Error())
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(DeleteResponse{Deleted: true})
}

// ==================== LIST EVENTS (Day/Week/Month) ====================

// listEventsHelper - общий хелпер для получения списка событий за период.
func (h *Handler) listEventsHelper(w http.ResponseWriter, r *http.Request, from, to time.Time) {
	ownerIDStr := r.URL.Query().Get("owner_id")
	if ownerIDStr == "" {
		http.Error(w, "owner_id query param is required", http.StatusBadRequest)
		return
	}

	var ownerID int64
	_, err := fmt.Sscanf(ownerIDStr, "%d", &ownerID)
	if err != nil || ownerID == 0 {
		http.Error(w, "invalid owner_id", http.StatusBadRequest)
		return
	}

	events, err := h.app.ListEvents(r.Context(), ownerID, from, to)
	if err != nil {
		h.logger.Error("ListEvents failed: " + err.Error())
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ListEventsResponse{
		Events: events,
		Total:  len(events),
	})
}

// ListEventsForDay обрабатывает GET /api/v1/calendar/events/day?owner_id=...&date=...
func (h *Handler) ListEventsForDay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		http.Error(w, "date query param is required", http.StatusBadRequest)
		return
	}

	date, err := parseRFC3339(dateStr)
	if err != nil || date == nil {
		http.Error(w, "invalid date format (use RFC3339)", http.StatusBadRequest)
		return
	}

	// Диапазон: весь день
	from := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	to := from.Add(24*time.Hour - time.Nanosecond)

	h.listEventsHelper(w, r, from, to)
}

// ListEventsForWeek обрабатывает GET /api/v1/calendar/events/week?owner_id=...&date=...
func (h *Handler) ListEventsForWeek(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		http.Error(w, "date query param is required", http.StatusBadRequest)
		return
	}

	date, err := parseRFC3339(dateStr)
	if err != nil || date == nil {
		http.Error(w, "invalid date format (use RFC3339)", http.StatusBadRequest)
		return
	}

	// Диапазон: неделя (пн–вс)
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := date.AddDate(0, 0, -(weekday - 1))
	from := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, date.Location())
	to := from.Add(7*24*time.Hour - time.Nanosecond)

	h.listEventsHelper(w, r, from, to)
}

// ListEventsForMonth обрабатывает GET /api/v1/calendar/events/month?owner_id=...&date=...
func (h *Handler) ListEventsForMonth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		http.Error(w, "date query param is required", http.StatusBadRequest)
		return
	}

	date, err := parseRFC3339(dateStr)
	if err != nil || date == nil {
		http.Error(w, "invalid date format (use RFC3339)", http.StatusBadRequest)
		return
	}

	// Диапазон: месяц
	from := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
	to := from.AddDate(0, 1, 0).Add(-time.Nanosecond)

	h.listEventsHelper(w, r, from, to)
}
