-- +goose Up
-- Архив заметок: архивная заметка скрывается из основного списка и групп,
-- видна в отдельном фильтре «Архив» у владельца; возвращается оттуда же.
ALTER TABLE public.notes ADD COLUMN archived boolean NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE public.notes DROP COLUMN IF EXISTS archived;
