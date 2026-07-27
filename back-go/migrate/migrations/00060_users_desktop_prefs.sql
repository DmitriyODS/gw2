-- +goose Up
-- Личные настройки рабочего стола (десктопный каркас): закреплённые в панели
-- задач разделы, размер плиток меню «Пуск», обои. Хранятся на сервере, чтобы
-- переезжать между устройствами пользователя; структуру ведёт фронт.
ALTER TABLE users
    ADD COLUMN desktop_prefs JSONB NOT NULL DEFAULT '{}'::jsonb;

-- +goose Down
ALTER TABLE users
    DROP COLUMN IF EXISTS desktop_prefs;
