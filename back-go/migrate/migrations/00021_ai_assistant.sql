-- +goose Up
-- Хранилище диалога ИИ-ассистента (расширение aisvc, Сущность 3 плана):
-- один диалог на пользователя+компанию, история — плоский лог сообщений.
CREATE TABLE ai_assistant_conversations (
    id bigserial PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_id integer NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, company_id)
);

CREATE TABLE ai_assistant_messages (
    id bigserial PRIMARY KEY,
    conversation_id bigint NOT NULL REFERENCES ai_assistant_conversations(id) ON DELETE CASCADE,
    role text NOT NULL CHECK (role IN ('user', 'assistant')),
    text text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_ai_assistant_messages_conv ON ai_assistant_messages (conversation_id, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_ai_assistant_messages_conv;
DROP TABLE ai_assistant_messages;
DROP TABLE ai_assistant_conversations;
