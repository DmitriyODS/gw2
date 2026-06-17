-- +goose Up
-- Редактирование сообщений: отметка времени последней правки. NULL — сообщение
-- не редактировалось (клиенты по наличию значения показывают пометку «изменено»).
ALTER TABLE messages ADD COLUMN edited_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE messages DROP COLUMN edited_at;
