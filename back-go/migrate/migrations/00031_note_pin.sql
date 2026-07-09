-- +goose Up
-- Закрепление заметок: закреплённая (pinned_at IS NOT NULL) идёт первой в
-- списках владельца (свежезакреплённая — выше). NULL — не закреплена.
ALTER TABLE public.notes ADD COLUMN pinned_at timestamptz;

-- +goose Down
ALTER TABLE public.notes DROP COLUMN IF EXISTS pinned_at;
