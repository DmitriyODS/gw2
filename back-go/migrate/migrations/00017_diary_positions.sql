-- +goose Up
-- Ручной порядок записей внутри дня (перетаскивание в модалке дня).
-- 0 — «не упорядочено вручную»: такие записи сортируются по времени после
-- упорядоченных; reorder проставляет 1..N.
ALTER TABLE public.diary_records
    ADD COLUMN position integer NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE public.diary_records DROP COLUMN position;
