-- +goose Up
-- Ответы на комментарии портала (дерево произвольной вложенности) и лайки.
--
-- reply_to_id — родитель в том же посте; ON DELETE CASCADE: удаление
-- комментария уносит всю ветку ответов под ним (иначе осиротевшие ответы
-- висят без контекста).
ALTER TABLE public.portal_comments
    ADD COLUMN reply_to_id bigint REFERENCES public.portal_comments(id) ON DELETE CASCADE;

CREATE INDEX portal_comments_reply_to_idx ON public.portal_comments(reply_to_id);

-- Лайк — toggle, одна строка на пару (комментарий, пользователь).
CREATE TABLE public.portal_comment_likes (
    comment_id bigint NOT NULL REFERENCES public.portal_comments(id) ON DELETE CASCADE,
    user_id    bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (comment_id, user_id)
);

-- +goose Down
DROP TABLE public.portal_comment_likes;
DROP INDEX public.portal_comments_reply_to_idx;
ALTER TABLE public.portal_comments DROP COLUMN reply_to_id;
