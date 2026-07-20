-- +goose Up
-- Персональное оформление ленты корпоративного портала: градиент фона + узор
-- + своя картинка с размытием (тот же рецепт, что у фона чатов, см. 00053).
-- Настройка личная (одна на пользователя) и синхронизируется между устройствами.
-- recipe — непрозрачный для бэкенда JSON (форму владеет фронт).
CREATE TABLE portal_backgrounds (
    user_id    BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    recipe     JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS portal_backgrounds;
