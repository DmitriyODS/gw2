-- +goose Up
-- Персональная статистика (ProfileStats/UserTasksDetail) фильтрует юниты по
-- пользователю и диапазону дат — одноколоночного idx_units_user мало.
CREATE INDEX idx_units_user_start ON public.units (user_id, datetime_start);

-- +goose Down
DROP INDEX IF EXISTS public.idx_units_user_start;
