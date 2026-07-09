-- +goose Up
-- Цвет заметки — тот же набор из 8 цветов-тегов, что у задач (--tag-* на
-- фронте). Заметка личная, поэтому цвет хранится прямо в notes ('' — без цвета).
ALTER TABLE public.notes ADD COLUMN color varchar(16) NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE public.notes DROP COLUMN IF EXISTS color;
