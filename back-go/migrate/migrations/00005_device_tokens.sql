-- +goose Up
-- Токены устройств для пуш-уведомлений (FCM). Один пользователь — много
-- устройств; токен глобально уникален (PK). На отзыв доступа/удаление
-- пользователя токены чистятся каскадом.
CREATE TABLE device_tokens (
    token      TEXT PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform   TEXT NOT NULL DEFAULT 'android',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_device_tokens_user ON device_tokens (user_id);

-- +goose Down
DROP TABLE device_tokens;
