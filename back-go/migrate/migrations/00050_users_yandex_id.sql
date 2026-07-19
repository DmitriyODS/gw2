-- +goose Up
-- Вход через Яндекс ID: постоянный идентификатор аккаунта Яндекса,
-- привязанный к пользователю (NULL — не привязан).
ALTER TABLE users ADD COLUMN yandex_id TEXT;
CREATE UNIQUE INDEX users_yandex_id_key ON users (yandex_id) WHERE yandex_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS users_yandex_id_key;
ALTER TABLE users DROP COLUMN IF EXISTS yandex_id;
