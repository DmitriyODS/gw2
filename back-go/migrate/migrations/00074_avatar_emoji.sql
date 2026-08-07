-- +goose Up
-- Аватар-значок: вместо фотографии человек может поставить эмодзи.
--
-- Приоритет показа: загруженный файл (avatar_path) → эмодзи → автоматический
-- identicon. Отдаёт всё это ОДНА ручка /api/users/<id>/identicon, поэтому
-- значок появляется всюду, где показывается аватар, без правок в интерфейсе.
ALTER TABLE public.users ADD COLUMN avatar_emoji varchar(32);

-- +goose Down
ALTER TABLE public.users DROP COLUMN IF EXISTS avatar_emoji;
