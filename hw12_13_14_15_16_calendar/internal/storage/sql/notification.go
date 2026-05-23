package sqlstorage

import (
	"context"
	"time"

	"github.com/ivvklimov/otus-go-professional/hw12_13_14_15_16_calendar/internal/storage"
)

// FetchPendingNotifications выбирает события, которые нужно опубликовать в очередь.
// Критерии:
// 1. notify_at <= NOW() (время напоминания пришло)
// 2. notify_published_at IS NULL (еще не публиковали в RabbitMQ)
// 3. date_start >= NOW() - 1 year (событие не архивное).
func (s *Storage) FetchPendingNotifications(ctx context.Context, limit int) ([]storage.Notification, error) {
	query := `
		SELECT id, title, date_start, owner_id
		FROM events
		WHERE notify_at IS NOT NULL
		  AND notify_at <= NOW()
		  AND notify_published_at IS NULL  -- Ищем события без даты публикации
		  AND date_start >= NOW() - INTERVAL '1 year'
		ORDER BY notify_at ASC
		LIMIT $1`

	var notifications []storage.Notification
	err := s.db.SelectContext(ctx, &notifications, query, limit)
	if err != nil {
		return nil, err
	}
	return notifications, nil
}

// MarkNotificationsSent помечает события как опубликованные в очереди.
// Устанавливает текущее время в notify_published_at.
func (s *Storage) MarkNotificationsSent(ctx context.Context, eventIDs []string) error {
	if len(eventIDs) == 0 {
		return nil
	}

	query := `
		UPDATE events 
		SET notify_published_at = NOW(), -- Ставим метку времени публикации
		    updated_at = NOW() 
		WHERE id = ANY($1)`

	_, err := s.db.ExecContext(ctx, query, eventIDs)
	return err
}

// DeleteOldEvents удаляет события, которые завершились более чем на olderThan назад.
func (s *Storage) DeleteOldEvents(ctx context.Context, olderThan time.Duration) (int64, error) {
	// Вычисляем пороговое время в Go: "сейчас минус заданный интервал"
	cutoff := time.Now().Add(-olderThan)

	query := `DELETE FROM events WHERE date_end < $1`
	res, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
