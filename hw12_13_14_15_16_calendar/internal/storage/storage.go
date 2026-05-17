package storage

import (
	"context"
	"time"
)

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
}
