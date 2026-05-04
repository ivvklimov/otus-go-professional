-- +goose Up
-- +goose StatementBegin

-- Расширения
CREATE EXTENSION IF NOT EXISTS "pgcrypto";    -- gen_random_uuid() для PG < 13
CREATE EXTENSION IF NOT EXISTS "btree_gist";  -- поддержка = в EXCLUDE USING gist

CREATE TABLE IF NOT EXISTS events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title           VARCHAR(255) NOT NULL,
    description     TEXT,
    owner_id        BIGINT NOT NULL,
    
    date_start      TIMESTAMPTZ NOT NULL,
    date_end        TIMESTAMPTZ NOT NULL,
    
    notify_at       TIMESTAMPTZ,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- 1. Дата окончания должна быть строго позже даты начала
    CONSTRAINT events_date_check CHECK (date_end > date_start),
    
    -- 2. Запрет пересечений интервалов для одного владельца
    -- Возвращает ошибку 23P01 (exclusion_violation) при конфликте
    CONSTRAINT events_no_overlap EXCLUDE USING gist (
        owner_id WITH =,
        tstzrange(date_start, date_end, '[)') WITH &&
    )
);

-- Индексы для ускорения поиска
CREATE INDEX idx_events_owner_period ON events (owner_id, date_start, date_end);
CREATE INDEX idx_events_notify_at ON events (notify_at) WHERE notify_at IS NOT NULL;
CREATE INDEX idx_events_owner_id ON events (owner_id);

-- Триггер для автоматического обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_updated_at
    BEFORE UPDATE ON events
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_update_updated_at ON events;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP INDEX IF EXISTS idx_events_owner_period;
DROP INDEX IF EXISTS idx_events_notify_at;
DROP INDEX IF EXISTS idx_events_owner_id;
DROP TABLE IF EXISTS events;
-- +goose StatementEnd
