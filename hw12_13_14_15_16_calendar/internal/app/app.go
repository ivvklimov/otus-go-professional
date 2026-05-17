package app

import (
	"context"
	"fmt"
	"time"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/logger"
	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage"
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
func (a *App) CreateEvent(ctx context.Context, title string, ownerID int64, description *string, dateStart, dateEnd time.Time, notifyAt *time.Time) (*storage.Event, error) {
	a.logger.Info("creating event: " + title)

	// Создаём событие с пустым ID - хранилище заполнит его (БД через RETURNING, memory - сама)
	evt := &storage.Event{
		ID:          "",
		Title:       title,
		OwnerID:     ownerID,
		Description: description,
		DateStart:   dateStart,
		DateEnd:     dateEnd,
		NotifyAt:    notifyAt,
	}

	// Вызываем хранилище - после этого evt.ID будет заполнен
	if err := a.storage.CreateEvent(ctx, evt); err != nil {
		return nil, err
	}

	a.logger.Info("event created: id=" + evt.ID)
	return evt, nil
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
func (a *App) ListEvents(ctx context.Context, ownerID int64, from, to time.Time) ([]storage.Event, error) {
	a.logger.Info(fmt.Sprintf("listing events for owner: %d", ownerID))
	return a.storage.ListEvents(ctx, ownerID, from, to)
}

// ==================== INTERFACE FOR TESTING ====================

// Service описывает контракт слоя приложения.
// Это позволяет подменять реализацию в юнит-тестах хэндлеров.
type Service interface {
	CreateEvent(ctx context.Context, title string, ownerID int64, description *string, dateStart, dateEnd time.Time, notifyAt *time.Time) (*storage.Event, error)
	UpdateEvent(ctx context.Context, id string, event storage.Event) error
	DeleteEvent(ctx context.Context, id string) error
	GetEvent(ctx context.Context, id string) (storage.Event, error)
	ListEvents(ctx context.Context, ownerID int64, from, to time.Time) ([]storage.Event, error)
}

// Ensure *App implements Service.
var _ Service = (*App)(nil)
