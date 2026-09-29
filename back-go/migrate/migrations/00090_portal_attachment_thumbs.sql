-- +goose Up
-- Миниатюры картинок портала: лента показывала оригиналы вложений, и на
-- телефоне каждый пост стоил мегабайты трафика и памяти. Миниатюра — отдельный
-- объект хранилища рядом с оригиналом (NULL — не картинка или она и так мала).
ALTER TABLE public.portal_attachments ADD COLUMN thumb_path text;

-- +goose Down
ALTER TABLE public.portal_attachments DROP COLUMN IF EXISTS thumb_path;
