-- +goose Up
-- Отдельный флаг видимости поля в карточке события календаря (виды «День»/«Неделя»).
-- show_in_table управляет плиткой/таблицей/экспортом, show_in_card — телом карточки.
ALTER TABLE public.calendar_fields
    ADD COLUMN show_in_card boolean NOT NULL DEFAULT true;

-- +goose Down
ALTER TABLE public.calendar_fields DROP COLUMN show_in_card;
