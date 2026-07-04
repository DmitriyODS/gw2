-- +goose Up
-- Адресный шаринг ежедневника: право отмечать записи выполненными.
-- Сценарий «руководитель раздаёт задачи»: владелец ведёт список, адресат
-- закрывает записи, не имея прав на структуру.
ALTER TABLE public.diary_user_shares
    ADD COLUMN can_check boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE public.diary_user_shares DROP COLUMN can_check;
