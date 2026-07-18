-- +goose Up
-- Иерархические папки заметок (как в Obsidian/Google Drive): у папки есть
-- родитель (parent_id NULL — корень), заметка лежит РОВНО в одной папке
-- (notes.folder_id NULL — корень). Папка личная (owner_id), как и заметка,
-- и не зависит от компании. Защита от циклов при переносе — в сервисе.
CREATE TABLE public.note_folders (
    id         bigserial PRIMARY KEY,
    owner_id   bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    parent_id  bigint REFERENCES public.note_folders(id) ON DELETE CASCADE,
    name       varchar(200) NOT NULL,
    color      varchar(16) NOT NULL DEFAULT '',
    position   integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX note_folders_owner_idx ON public.note_folders (owner_id, parent_id, position);

-- Папка заметки (NULL — корень/без папки). ON DELETE SET NULL: удаление папки
-- само не роняет заметки — сервис перед удалением переносит их в родителя.
ALTER TABLE public.notes ADD COLUMN folder_id bigint
    REFERENCES public.note_folders(id) ON DELETE SET NULL;
CREATE INDEX notes_folder_idx ON public.notes (folder_id);

-- +goose Down
ALTER TABLE public.notes DROP COLUMN IF EXISTS folder_id;
DROP TABLE IF EXISTS public.note_folders;
