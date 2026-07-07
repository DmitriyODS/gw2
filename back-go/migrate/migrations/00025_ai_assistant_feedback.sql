-- +goose Up
-- Провенанс ответа ИИ-ассистента: готовая строка «Данные: …» — какие
-- инструменты (и за какой период) реально дали факты для ответа. NULL —
-- инструменты не вызывались.
ALTER TABLE ai_assistant_messages ADD COLUMN sources text;

-- Обратная связь 👍/👎 по ответам ассистента: один голос на пару
-- (сообщение, пользователь), повторный голос заменяет прежний (upsert).
CREATE TABLE ai_assistant_feedback (
    id bigserial PRIMARY KEY,
    message_id bigint NOT NULL REFERENCES ai_assistant_messages(id) ON DELETE CASCADE,
    user_id bigint NOT NULL,
    verdict text NOT NULL,
    reason text,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (message_id, user_id)
);

-- +goose Down
DROP TABLE ai_assistant_feedback;
ALTER TABLE ai_assistant_messages DROP COLUMN sources;
