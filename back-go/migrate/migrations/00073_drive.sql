-- +goose Up
-- Раздел «Диск» (drivesvc): личное файловое хранилище с папками, корзиной и
-- шарингом — по устройству близнец заметок и досок.
--
-- Диск принадлежит ОДНОМУ пользователю и не зависит от компании: доступ
-- коллегам и компаниям выдаётся шарингом (как у заметок), а не владением.
-- Место занимают файлы диска в общей квоте владельца (billingsvc).

CREATE TABLE public.drive_folders (
    id         bigserial PRIMARY KEY,
    owner_id   bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    parent_id  bigint REFERENCES public.drive_folders(id) ON DELETE CASCADE,
    name       varchar(200) NOT NULL,
    color      varchar(16) NOT NULL DEFAULT '',
    -- Корзина: папка и всё её содержимое скрываются, но переживают отмену.
    -- Место освобождается только при окончательном удалении.
    deleted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX drive_folders_owner_idx ON public.drive_folders (owner_id, parent_id);
CREATE INDEX drive_folders_trash_idx ON public.drive_folders (owner_id, deleted_at);

CREATE TABLE public.drive_files (
    id          bigserial PRIMARY KEY,
    owner_id    bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    -- Папка файла (NULL — корень диска). ON DELETE SET NULL: удаление папки
    -- не роняет файлы — сервис сам решает, переносить их или отправлять в
    -- корзину вместе с ней.
    folder_id   bigint REFERENCES public.drive_folders(id) ON DELETE SET NULL,
    name        varchar(255) NOT NULL,
    -- storage_key — ключ объекта в хранилище (pkg/storage), он же хвост
    -- адреса /uploads/<key>. Уникален: один файл — один объект.
    storage_key varchar(300) NOT NULL UNIQUE,
    mime        varchar(120) NOT NULL DEFAULT '',
    size_bytes  bigint NOT NULL DEFAULT 0,
    starred     boolean NOT NULL DEFAULT FALSE,
    deleted_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX drive_files_owner_idx ON public.drive_files (owner_id, folder_id);
CREATE INDEX drive_files_trash_idx ON public.drive_files (owner_id, deleted_at);
-- «Недавние» и сортировка по дате.
CREATE INDEX drive_files_recent_idx ON public.drive_files (owner_id, updated_at DESC);
-- Поиск по имени файла — триграммный GIN, как у заметок и реестров.
CREATE INDEX drive_files_search_idx ON public.drive_files USING gin (name public.gin_trgm_ops);

-- Публичная ссылка на файл или папку (код-capability в адресе).
CREATE TABLE public.drive_shares (
    id         bigserial PRIMARY KEY,
    file_id    bigint REFERENCES public.drive_files(id) ON DELETE CASCADE,
    folder_id  bigint REFERENCES public.drive_folders(id) ON DELETE CASCADE,
    code       varchar(40) NOT NULL UNIQUE,
    created_by bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    -- Ссылка ведёт ровно на одну сущность.
    CONSTRAINT ck_drive_share_target CHECK (
        (file_id IS NOT NULL AND folder_id IS NULL)
        OR (file_id IS NULL AND folder_id IS NOT NULL)
    )
);

-- Адресный доступ: конкретному человеку или всей компании. Доступ по папке
-- КАСКАДИТ на её поддерево — как у заметок и досок.
CREATE TABLE public.drive_user_shares (
    id         bigserial PRIMARY KEY,
    file_id    bigint REFERENCES public.drive_files(id) ON DELETE CASCADE,
    folder_id  bigint REFERENCES public.drive_folders(id) ON DELETE CASCADE,
    user_id    bigint REFERENCES public.users(id) ON DELETE CASCADE,
    company_id bigint REFERENCES public.companies(id) ON DELETE CASCADE,
    can_edit   boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_drive_user_share_target CHECK (
        (file_id IS NOT NULL AND folder_id IS NULL)
        OR (file_id IS NULL AND folder_id IS NOT NULL)
    ),
    -- Адресат — либо человек, либо компания.
    CONSTRAINT ck_drive_user_share_who CHECK (
        (user_id IS NOT NULL AND company_id IS NULL)
        OR (user_id IS NULL AND company_id IS NOT NULL)
    )
);
CREATE INDEX drive_user_shares_user_idx ON public.drive_user_shares (user_id);
CREATE INDEX drive_user_shares_company_idx ON public.drive_user_shares (company_id);
-- Один адресат — одна строка на сущность (повторная выдача обновляет права).
CREATE UNIQUE INDEX drive_user_shares_file_user_idx
    ON public.drive_user_shares (file_id, user_id) WHERE file_id IS NOT NULL AND user_id IS NOT NULL;
CREATE UNIQUE INDEX drive_user_shares_file_company_idx
    ON public.drive_user_shares (file_id, company_id) WHERE file_id IS NOT NULL AND company_id IS NOT NULL;
CREATE UNIQUE INDEX drive_user_shares_folder_user_idx
    ON public.drive_user_shares (folder_id, user_id) WHERE folder_id IS NOT NULL AND user_id IS NOT NULL;
CREATE UNIQUE INDEX drive_user_shares_folder_company_idx
    ON public.drive_user_shares (folder_id, company_id) WHERE folder_id IS NOT NULL AND company_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS public.drive_user_shares;
DROP TABLE IF EXISTS public.drive_shares;
DROP TABLE IF EXISTS public.drive_files;
DROP TABLE IF EXISTS public.drive_folders;
