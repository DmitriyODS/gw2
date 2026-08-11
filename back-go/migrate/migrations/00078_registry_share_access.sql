-- +goose Up
-- Внешние ссылки на реестр двух видов: только просмотр и просмотр с правкой
-- записей. Уровень — свойство САМОЙ ссылки (а не реестра): владелец раздаёт
-- разным людям разные коды и отзывает их по отдельности.
--
-- Прежние ссылки остаются read-only — умолчание совпадает с их поведением.
ALTER TABLE public.registry_shares
    ADD COLUMN access text NOT NULL DEFAULT 'view'
        CHECK (access IN ('view', 'edit'));

-- +goose Down
ALTER TABLE public.registry_shares
    DROP COLUMN IF EXISTS access;
