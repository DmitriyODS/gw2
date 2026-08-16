-- +goose Up
-- Условное отображение в формах: раздел или вопрос показывается, только если на
-- вопрос-источник дан один из ожидаемых ответов.
--
-- Это НЕ ветвление: ветвление уводит на другую страницу целиком, а условие
-- прячет отдельный вопрос (или раздел) внутри того же маршрута. У вопроса
-- условие живёт в его config (JSONB уже есть), разделу нужны свои колонки.

ALTER TABLE public.form_sections
    -- Вопрос-источник: ссылка на form_questions той же формы. Ссылка «раздел →
    -- вопрос → раздел» образует цикл, поэтому проверку откладываем до коммита —
    -- восстановление бэкапа льёт таблицы одной транзакцией.
    ADD COLUMN visible_question_id bigint REFERENCES public.form_questions(id)
        ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED,
    -- Ожидаемые ответы; пустой список означает «любой непустой ответ».
    ADD COLUMN visible_values jsonb NOT NULL DEFAULT '[]'::jsonb;

-- +goose Down
ALTER TABLE public.form_sections
    DROP COLUMN IF EXISTS visible_values,
    DROP COLUMN IF EXISTS visible_question_id;
