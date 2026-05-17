package storage

import (
	"context"
	"time"
)

// Notification — общая структура для передачи данных о событии в очередь.
// Определяем здесь, чтобы интерфейс и все реализации могли его использовать.
// Notification — общая структура для передачи данных о событии в очередь.
type Notification struct {
	EventID     string    `json:"eventId" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description,omitempty" db:"description"`
	DateStart   time.Time `json:"dateStart" db:"date_start"`
	DateEnd     time.Time `json:"dateEnd" db:"date_end"`
	OwnerID     int64     `json:"ownerId" db:"owner_id"`
	NotifyAt    time.Time `json:"notifyAt" db:"notify_at"`
}

// Storage - интерфейс для работы с хранилищем событий.
// Любая реализация (in-memory, SQL) должна соответствовать этому контракту.
type Storage interface {
	// CreateEvent добавляет событие в хранилище.
	// Возвращает ErrDateBusy, если время занято.
	CreateEvent(ctx context.Context, event *Event) error

	// UpdateEvent изменяет существующее событие.
	UpdateEvent(ctx context.Context, id string, event Event) error

	// DeleteEvent удаляет событие по ID.
	DeleteEvent(ctx context.Context, id string) error

	// GetEvent получает событие по ID.
	GetEvent(ctx context.Context, id string) (Event, error)

	// ListEvents получает список событий владельца за указанный период.
	ListEvents(ctx context.Context, ownerID int64, from, to time.Time) ([]Event, error)

	// Notification methods (new for DZ #14)
	FetchPendingNotifications(ctx context.Context, limit int) ([]Notification, error)
	MarkNotificationsSent(ctx context.Context, eventIDs []string) error
	DeleteOldEvents(ctx context.Context, olderThan time.Duration) (int64, error)
}
