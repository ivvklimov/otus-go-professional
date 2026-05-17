package internalhttp

import (
	"time"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage"
)

// ==================== ЗАПРОСЫ ====================

// CreateEventRequest - тело POST-запроса на создание события.
type CreateEventRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	DateStart   string `json:"dateStart"` // RFC3339: "2026-05-10T10:00:00Z"
	DateEnd     string `json:"dateEnd"`
	OwnerID     int64  `json:"ownerId"`
	NotifyAt    string `json:"notifyAt,omitempty"`
}

// UpdateEventRequest - тело PUT-запроса на обновление события.
// Все поля опциональны - частичное обновление.
type UpdateEventRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	DateStart   *string `json:"dateStart,omitempty"`
	DateEnd     *string `json:"dateEnd,omitempty"`
	OwnerID     *int64  `json:"ownerId,omitempty"`
	NotifyAt    *string `json:"notifyAt,omitempty"`
}

// ==================== ОТВЕТЫ ====================

// EventResponse - обёртка для ответа с одним событием.
type EventResponse struct {
	Event storage.Event `json:"event"`
}

// ListEventsResponse - ответ со списком событий.
type ListEventsResponse struct {
	Events []storage.Event `json:"events"`
	Total  int             `json:"total"`
}

// DeleteResponse - ответ на удаление.
type DeleteResponse struct {
	Deleted bool `json:"deleted"`
}

// ==================== ХЕЛПЕРЫ ====================

// parseRFC3339 парсит строку в time.Time с проверкой на пустое значение.
func parseRFC3339(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
