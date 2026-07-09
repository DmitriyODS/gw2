-- +goose Up
-- Портал больше не публикует системные посты (gRPC CreateSystemPost и
-- пост-поздравления petsvc при эволюции грувика удалены): посты создают
-- только сотрудники. Существующие системные посты удаляются (комментарии/
-- реакции/вложения подчистят каскады), колонка system_kind не нужна.
DELETE FROM public.portal_posts WHERE system_kind IS NOT NULL;
ALTER TABLE public.portal_posts DROP COLUMN system_kind;

-- +goose Down
ALTER TABLE public.portal_posts ADD COLUMN system_kind text;
