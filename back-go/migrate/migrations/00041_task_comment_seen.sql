-- +goose Up
-- Отметка «прочитано» комментариев задачи пользователем: last_seen_at — момент,
-- когда пользователь последний раз открывал вкладку комментариев задачи. Новые
-- комментарии = чужие с created_at > last_seen_at (нет строки → все чужие новые).
CREATE TABLE public.task_comment_seen (
    user_id integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    task_id integer NOT NULL REFERENCES public.tasks(id) ON DELETE CASCADE,
    last_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (user_id, task_id)
);

-- +goose Down
DROP TABLE public.task_comment_seen;
