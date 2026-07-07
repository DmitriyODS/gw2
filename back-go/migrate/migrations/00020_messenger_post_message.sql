-- +goose Up
-- Пересылка поста портала в мессенджер: новый вид сообщения kind='post'
-- (аналог kind='task'/'call'). В отличие от задачи (живой JOIN на tasks),
-- превью поста — ЗАМОРОЖЕННЫЙ снапшот на момент пересылки (title/excerpt/
-- cover_url приходят от portalsvc в gRPC-запросе CreatePostMessage и
-- сохраняются как есть, без обращения к таблицам portal_*) — так мессенджер
-- не завязывается на схему portalsvc. post_id — голый BIGINT без FK на
-- portal_posts: кросс-сервисная ссылка против архитектуры, а ON DELETE SET
-- NULL стирал бы замороженное превью при удалении поста.
ALTER TABLE public.messages
    ADD COLUMN post_id bigint,
    ADD COLUMN post_title text,
    ADD COLUMN post_excerpt text,
    ADD COLUMN post_cover_url text;

CREATE INDEX idx_msg_post ON public.messages (post_id);

-- +goose Down
DROP INDEX public.idx_msg_post;
ALTER TABLE public.messages
    DROP COLUMN post_cover_url,
    DROP COLUMN post_excerpt,
    DROP COLUMN post_title,
    DROP COLUMN post_id;
