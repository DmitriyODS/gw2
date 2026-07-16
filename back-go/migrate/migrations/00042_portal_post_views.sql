-- +goose Up
-- Просмотры постов портала: одна строка на пару (пост, пользователь) —
-- количество просмотров поста = число уникальных зрителей. Отметка ставится,
-- когда карточка поста попадает в поле зрения читателя.
CREATE TABLE public.portal_post_views (
    post_id   bigint NOT NULL REFERENCES public.portal_posts(id) ON DELETE CASCADE,
    user_id   bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    viewed_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (post_id, user_id)
);

-- +goose Down
DROP TABLE public.portal_post_views;
