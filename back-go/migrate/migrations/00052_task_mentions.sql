-- +goose Up
-- Упоминания людей в комментариях к задаче (@логин). Каждое упоминание —
-- строка; непрочитанные (seen_at IS NULL) дают бейдж на карточке задачи у
-- упомянутого. Автор комментария и упомянутый чистятся каскадом.
CREATE TABLE task_mentions (
    id         BIGSERIAL PRIMARY KEY,
    task_id    BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    comment_id BIGINT NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    seen_at    TIMESTAMPTZ
);

-- Батч-бейдж «сколько непрочитанных упоминаний у пользователя по задачам».
CREATE INDEX idx_task_mentions_unseen
    ON task_mentions (user_id, task_id)
    WHERE seen_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS task_mentions;
