-- +goose Up
-- Миниатюра картинки-вложения: в ленте чата грузится уменьшенное превью
-- (thumb_path), исходник — только по клику. NULL — превью нет (не картинка
-- или не удалось сжать), клиент показывает оригинал.
ALTER TABLE public.message_attachments ADD COLUMN thumb_path text;

-- +goose Down
ALTER TABLE public.message_attachments DROP COLUMN thumb_path;
