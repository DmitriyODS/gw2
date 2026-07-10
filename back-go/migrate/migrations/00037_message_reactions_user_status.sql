-- +goose Up
-- Реакции-эмодзи на сообщения мессенджера: many-to-many (несколько
-- пользователей × несколько эмодзи на сообщение), toggle по первичному ключу.
CREATE TABLE public.message_reactions (
    message_id integer NOT NULL REFERENCES public.messages(id) ON DELETE CASCADE,
    user_id integer NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    emoji text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (message_id, user_id, emoji)
);

-- Пользовательский статус (мессенджер): эмодзи + короткий текст.
ALTER TABLE public.users ADD COLUMN status_emoji text;
ALTER TABLE public.users ADD COLUMN status_text character varying(80);

-- +goose Down
ALTER TABLE public.users DROP COLUMN status_text;
ALTER TABLE public.users DROP COLUMN status_emoji;
DROP TABLE IF EXISTS public.message_reactions;
