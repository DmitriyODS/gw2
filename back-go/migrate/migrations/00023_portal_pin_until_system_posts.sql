-- +goose Up
-- Портал (portalsvc):
--  1) pinned_until — закрепление поста с автоистечением (NULL = бессрочно).
--     Истёкший пин (pinned_until <= now()) везде трактуется как незакреплённый
--     ФИЛЬТРОМ в выборках (лениво не чистим: read-путь без записи, история
--     закрепления сохраняется, повторный Pin просто перезапишет колонки).
--  2) system_kind — системные посты ('pet_evolved' и будущие; NULL = обычный
--     пост, создаётся сервисами по gRPC portal.v1.PortalService).
--  3) Индекс под keyset-пагинацию хронологии ленты
--     ORDER BY created_at DESC, id DESC в скоупе компании.
ALTER TABLE public.portal_posts ADD COLUMN pinned_until timestamptz;
ALTER TABLE public.portal_posts ADD COLUMN system_kind text;
CREATE INDEX portal_posts_feed_idx ON public.portal_posts (company_id, created_at DESC, id DESC);

-- +goose Down
DROP INDEX public.portal_posts_feed_idx;
ALTER TABLE public.portal_posts DROP COLUMN system_kind;
ALTER TABLE public.portal_posts DROP COLUMN pinned_until;
