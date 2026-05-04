package app

import (
	"context"
	"time"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_calendar/internal/logger"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_calendar/internal/storage"
)

// App представляет основную логику приложения (бизнес-слой).
type App struct {
	logger  *logger.Logger
	storage storage.Storage
}

// New создаёт новый экземпляр приложения с внедрёнными зависимостями.
func New(logger *logger.Logger, storage storage.Storage) *App {
	return &App{
		logger:  logger,
		storage: storage,
	}
}

// CreateEvent создаёт новое событие.
func (a *App) CreateEvent(ctx context.Context, id, title string) error {
	a.logger.Info("creating event: " + title)

	evt := storage.Event{
		ID:    id,
		Title: title,
	}

	return a.storage.CreateEvent(ctx, &evt)
}

// UpdateEvent обновляет существующее событие по ID.
func (a *App) UpdateEvent(ctx context.Context, id string, event storage.Event) error {
	a.logger.Info("updating event: " + id)
	return a.storage.UpdateEvent(ctx, id, event)
}

// DeleteEvent удаляет событие по ID.
func (a *App) DeleteEvent(ctx context.Context, id string) error {
	a.logger.Info("deleting event: " + id)
	return a.storage.DeleteEvent(ctx, id)
}

// GetEvent возвращает событие по его ID.
func (a *App) GetEvent(ctx context.Context, id string) (storage.Event, error) {
	a.logger.Info("getting event: " + id)
	return a.storage.GetEvent(ctx, id)
}

// ListEvents возвращает список событий для владельца в заданном периоде.
func (a *App) ListEvents(ctx context.Context, ownerID string, from, to time.Time) ([]storage.Event, error) {
	a.logger.Info("listing events for owner: " + ownerID)
	return a.storage.ListEvents(ctx, ownerID, from, to)
}
