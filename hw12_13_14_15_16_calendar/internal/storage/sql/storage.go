package sqlstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_calendar/internal/storage"
)

// Storage — реализация интерфейса storage.Storage на PostgreSQL.
type Storage struct {
	db *sqlx.DB
}

// New создаёт новое хранилище с подключением к БД.
func New(db *sqlx.DB) *Storage {
	return &Storage{db: db}
}

// CreateEvent добавляет событие в БД.
// event — указатель, чтобы мы могли записать туда сгенерированный ID.
func (s *Storage) CreateEvent(ctx context.Context, event *storage.Event) error {
	// Вставляем данные БЕЗ id. База сгенерирует его сама.
	query := `
		INSERT INTO events (
			title, description, owner_id,
			date_start, date_end, notify_at,
			created_at, updated_at
		) VALUES (
			:title, :description, :owner_id,
			:date_start, :date_end, :notify_at,
			NOW(), NOW()
		) RETURNING id
	`

	row, err := s.db.NamedQueryContext(ctx, query, event)
	if err != nil {
		if isConflictError(err) {
			return storage.ErrDateBusy
		}
		return fmt.Errorf("create event: %w", err)
	}
	defer row.Close()

	// Записываем сгенерированный БД ID обратно в структуру
	if row.Next() {
		if err := row.Scan(&event.ID); err != nil {
			return fmt.Errorf("scan returned id: %w", err)
		}
	}
	return nil
}

// UpdateEvent изменяет существующее событие.
func (s *Storage) UpdateEvent(ctx context.Context, id string, event storage.Event) error {
	// Сначала проверим, что событие существует и принадлежит владельцу
	exists, err := s.eventExists(ctx, id, event.OwnerID)
	if err != nil {
		return fmt.Errorf("check event exists: %w", err)
	}
	if !exists {
		return storage.ErrNotFound
	}

	query := `
		UPDATE events SET
			title = :title,
			description = :description,
			owner_id = :owner_id,
			date_start = :date_start,
			date_end = :date_end,
			notify_at = :notify_at,
			updated_at = NOW()
		WHERE id = :id
	`

	event.ID = id // Убедимся, что ID в структуре совпадает с параметром
	_, err = s.db.NamedExecContext(ctx, query, event)
	if err != nil {
		if isConflictError(err) {
			return storage.ErrDateBusy
		}
		return fmt.Errorf("update event: %w", err)
	}
	return nil
}

// DeleteEvent удаляет событие по ID.
func (s *Storage) DeleteEvent(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM events WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// GetEvent получает событие по ID.
func (s *Storage) GetEvent(ctx context.Context, id string) (storage.Event, error) {
	var event storage.Event
	// Явно перечисляем колонки, которые есть в структуре Event
	query := `
		SELECT id, title, description, owner_id, 
		       date_start, date_end, notify_at,
		       created_at, updated_at
		FROM events 
		WHERE id = $1
	`

	err := s.db.GetContext(ctx, &event, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Event{}, storage.ErrNotFound
		}
		return storage.Event{}, fmt.Errorf("get event: %w", err)
	}
	return event, nil
}

// ListEvents получает список событий владельца за период.
func (s *Storage) ListEvents(ctx context.Context, ownerID string, from, to time.Time) ([]storage.Event, error) {
	var events []storage.Event

	// Явно перечисляем колонки
	query := `
		SELECT id, title, description, owner_id,
		       date_start, date_end, notify_at,
		       created_at, updated_at
		FROM events 
		WHERE owner_id = $1 
			AND date_start < $2 
			AND date_end > $3
		ORDER BY date_start ASC
	`

	err := s.db.SelectContext(ctx, &events, query, ownerID, to, from)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	if events == nil {
		events = []storage.Event{}
	}
	return events, nil
}

// eventExists проверяет существование события с указанным ID и ownerID.
func (s *Storage) eventExists(ctx context.Context, id, ownerID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM events WHERE id = $1 AND owner_id = $2)`
	err := s.db.GetContext(ctx, &exists, query, id, ownerID)
	return exists, err
}

// isConflictError определяет, является ли ошибка конфликтом дат.
func isConflictError(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		// 23P01 = exclusion_violation (EXCLUDE constraint)
		// 23505 = unique_violation
		// 23514 = check_violation
		return pqErr.Code == "23P01" || pqErr.Code == "23505" || pqErr.Code == "23514"
	}
	return false
}
