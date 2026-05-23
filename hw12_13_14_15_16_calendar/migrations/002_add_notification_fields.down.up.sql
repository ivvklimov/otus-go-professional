-- +goose Up
-- +goose StatementBegin
-- Заменяем notify_sent на два поля для отслеживания этапов доставки
ALTER TABLE events ADD COLUMN IF NOT EXISTS notify_published_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE events ADD COLUMN IF NOT EXISTS notify_delivered_at TIMESTAMP WITH TIME ZONE;

-- Индекс для быстрого поиска событий, готовых к публикации
-- (notify_at настал, но в очередь ещё не положено)
DROP INDEX IF EXISTS idx_events_notify_pending;
CREATE INDEX idx_events_notify_pending ON events (notify_at) WHERE notify_published_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_events_notify_pending;
ALTER TABLE events DROP COLUMN IF EXISTS notify_published_at;
ALTER TABLE events DROP COLUMN IF EXISTS notify_delivered_at;
-- +goose StatementEnd
