-- +goose Up
-- Группы заметок (плоские папки-фильтры many-to-many) переезжают в ТЕГИ — та же
-- семантика cross-cutting-меток, но с цветом из палитры --tag-*. Иерархия теперь
-- отдельно (note_folders). Данные пользователей сохраняются переименованием.
ALTER TABLE public.note_groups RENAME TO note_tags;
ALTER TABLE public.note_group_items RENAME TO note_tag_items;
ALTER TABLE public.note_tag_items RENAME COLUMN group_id TO tag_id;
ALTER TABLE public.note_tags ADD COLUMN color varchar(16) NOT NULL DEFAULT '';

ALTER INDEX IF EXISTS note_groups_owner_idx RENAME TO note_tags_owner_idx;
ALTER INDEX IF EXISTS note_group_items_group_idx RENAME TO note_tag_items_tag_idx;

-- +goose Down
ALTER INDEX IF EXISTS note_tag_items_tag_idx RENAME TO note_group_items_group_idx;
ALTER INDEX IF EXISTS note_tags_owner_idx RENAME TO note_groups_owner_idx;
ALTER TABLE public.note_tags DROP COLUMN IF EXISTS color;
ALTER TABLE public.note_tag_items RENAME COLUMN tag_id TO group_id;
ALTER TABLE public.note_tag_items RENAME TO note_group_items;
ALTER TABLE public.note_tags RENAME TO note_groups;
