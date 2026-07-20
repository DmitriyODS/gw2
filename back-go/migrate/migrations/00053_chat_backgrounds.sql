-- +goose Up
-- Персональное оформление чатов мессенджера: градиент фона + узор-трафарет.
-- Настройка личная (у каждого пользователя своя) и синхронизируется между
-- устройствами. Строка с conversation_id IS NULL — общий дефолт пользователя,
-- строки с конкретным conversation_id — переопределение для отдельного чата.
-- recipe — непрозрачный для бэкенда JSON (форму рецепта владеет фронт:
-- пресет/пятна градиента, ключ узора, насыщённость).
CREATE TABLE chat_backgrounds (
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE,
    recipe          JSONB NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Один общий дефолт на пользователя.
CREATE UNIQUE INDEX chat_backgrounds_default_uniq
    ON chat_backgrounds (user_id)
    WHERE conversation_id IS NULL;

-- Одно переопределение на пару (пользователь, чат).
CREATE UNIQUE INDEX chat_backgrounds_conv_uniq
    ON chat_backgrounds (user_id, conversation_id)
    WHERE conversation_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS chat_backgrounds;
