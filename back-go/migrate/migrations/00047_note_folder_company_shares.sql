-- +goose Up
-- Шаринг папок и заметок целым компаниям, а также папок — конкретным
-- пользователям. Заметки/папки кросс-компанийные (привязаны к владельцу), при
-- шаринге с компанией виден всем её текущим/будущим сотрудникам. company_name —
-- денормализация для отображения (имя компании на момент шаринга). can_edit —
-- чтение+редактирование, иначе только чтение. Доступ по папке КАСКАДИТ на всё
-- её поддерево (эффективный доступ считается в сервисе подъёмом по parent_id).

-- Заметка → компании.
CREATE TABLE public.note_company_shares (
    note_id      bigint NOT NULL REFERENCES public.notes(id) ON DELETE CASCADE,
    company_id   bigint NOT NULL,
    company_name varchar(200) NOT NULL DEFAULT '',
    can_edit     boolean NOT NULL DEFAULT FALSE,
    shared_by    bigint NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (note_id, company_id)
);
CREATE INDEX note_company_shares_company_idx ON public.note_company_shares (company_id);

-- Папка → пользователю (доступ каскадит на всё поддерево).
CREATE TABLE public.folder_user_shares (
    folder_id  bigint NOT NULL REFERENCES public.note_folders(id) ON DELETE CASCADE,
    user_id    bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    can_edit   boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (folder_id, user_id)
);
CREATE INDEX folder_user_shares_user_idx ON public.folder_user_shares (user_id);

-- Папка → компании.
CREATE TABLE public.folder_company_shares (
    folder_id    bigint NOT NULL REFERENCES public.note_folders(id) ON DELETE CASCADE,
    company_id   bigint NOT NULL,
    company_name varchar(200) NOT NULL DEFAULT '',
    can_edit     boolean NOT NULL DEFAULT FALSE,
    shared_by    bigint NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (folder_id, company_id)
);
CREATE INDEX folder_company_shares_company_idx ON public.folder_company_shares (company_id);

-- +goose Down
DROP TABLE IF EXISTS public.folder_company_shares;
DROP TABLE IF EXISTS public.folder_user_shares;
DROP TABLE IF EXISTS public.note_company_shares;
