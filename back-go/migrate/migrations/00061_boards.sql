-- +goose Up
-- Раздел «Доски»: личные доски рисования пользователя (сцена холста в JSONB) с
-- тем же набором возможностей, что и у заметок — иерархические папки, теги-метки,
-- закрепление, архив, публичные ссылки, адресный шаринг пользователям и целым
-- компаниям (доступ по папке каскадит на поддерево) и личный оверлей адресата.
-- Доска принадлежит ОДНОМУ пользователю (owner_id) и не зависит от компании
-- (кросс-компанийная, как заметка). text_content — плоский текст надписей и
-- стикеров, пересчитывается сервером из scene при сохранении (сквозной поиск).
-- preview_path — миниатюра холста в хранилище (снимает клиент), для плиток.

CREATE TABLE public.board_folders (
    id         bigserial PRIMARY KEY,
    owner_id   bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    parent_id  bigint REFERENCES public.board_folders(id) ON DELETE CASCADE,
    name       varchar(200) NOT NULL,
    color      varchar(16) NOT NULL DEFAULT '',
    position   integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX board_folders_owner_idx ON public.board_folders (owner_id, parent_id, position);

CREATE TABLE public.boards (
    id           bigserial PRIMARY KEY,
    owner_id     bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    folder_id    bigint REFERENCES public.board_folders(id) ON DELETE SET NULL,
    title        varchar(300) NOT NULL DEFAULT '',
    color        varchar(16) NOT NULL DEFAULT '',
    archived     boolean NOT NULL DEFAULT FALSE,
    pinned_at    timestamptz,
    scene        jsonb NOT NULL DEFAULT '{"version":1,"background":"grid","objects":[]}',
    text_content text NOT NULL DEFAULT '',
    preview_path varchar(300) NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX boards_owner_idx ON public.boards (owner_id, updated_at DESC);
CREATE INDEX boards_folder_idx ON public.boards (folder_id);
-- Триграммный индекс под сквозной ILIKE-поиск по названию и надписям холста.
CREATE INDEX boards_search_idx ON public.boards
    USING gin ((title || ' ' || text_content) public.gin_trgm_ops);

-- Теги-метки досок (many-to-many, цвет из палитры --tag-*).
CREATE TABLE public.board_tags (
    id         bigserial PRIMARY KEY,
    owner_id   bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    name       varchar(100) NOT NULL,
    color      varchar(16) NOT NULL DEFAULT '',
    position   integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX board_tags_owner_idx ON public.board_tags (owner_id, position);

CREATE TABLE public.board_tag_items (
    board_id bigint NOT NULL REFERENCES public.boards(id) ON DELETE CASCADE,
    tag_id   bigint NOT NULL REFERENCES public.board_tags(id) ON DELETE CASCADE,
    PRIMARY KEY (board_id, tag_id)
);
CREATE INDEX board_tag_items_tag_idx ON public.board_tag_items (tag_id);

-- Публичные ссылки на доску (без авторизации): код в URL — capability.
CREATE TABLE public.board_shares (
    id         bigserial PRIMARY KEY,
    board_id   bigint NOT NULL REFERENCES public.boards(id) ON DELETE CASCADE,
    code       varchar(40) NOT NULL UNIQUE,
    access     varchar(8) NOT NULL CHECK (access IN ('view', 'edit')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX board_shares_board_idx ON public.board_shares (board_id);

-- Адресный шаринг доски: конкретному пользователю и целой компании.
CREATE TABLE public.board_user_shares (
    board_id   bigint NOT NULL REFERENCES public.boards(id) ON DELETE CASCADE,
    user_id    bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    can_edit   boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (board_id, user_id)
);
CREATE INDEX board_user_shares_user_idx ON public.board_user_shares (user_id);

CREATE TABLE public.board_company_shares (
    board_id     bigint NOT NULL REFERENCES public.boards(id) ON DELETE CASCADE,
    company_id   bigint NOT NULL,
    company_name varchar(200) NOT NULL DEFAULT '',
    can_edit     boolean NOT NULL DEFAULT FALSE,
    shared_by    bigint NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (board_id, company_id)
);
CREATE INDEX board_company_shares_company_idx ON public.board_company_shares (company_id);

-- Шаринг папки досок (каскадит на всё поддерево).
CREATE TABLE public.board_folder_user_shares (
    folder_id  bigint NOT NULL REFERENCES public.board_folders(id) ON DELETE CASCADE,
    user_id    bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    can_edit   boolean NOT NULL DEFAULT FALSE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (folder_id, user_id)
);
CREATE INDEX board_folder_user_shares_user_idx ON public.board_folder_user_shares (user_id);

CREATE TABLE public.board_folder_company_shares (
    folder_id    bigint NOT NULL REFERENCES public.board_folders(id) ON DELETE CASCADE,
    company_id   bigint NOT NULL,
    company_name varchar(200) NOT NULL DEFAULT '',
    can_edit     boolean NOT NULL DEFAULT FALSE,
    shared_by    bigint NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (folder_id, company_id)
);
CREATE INDEX board_folder_company_shares_company_idx ON public.board_folder_company_shares (company_id);

-- Личный оверлей адресата: чужую доску/папку можно разложить по СВОИМ папкам и
-- отправить в личный архив, не трогая владельца.
CREATE TABLE public.board_recipient_state (
    user_id    bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    board_id   bigint NOT NULL REFERENCES public.boards(id) ON DELETE CASCADE,
    folder_id  bigint REFERENCES public.board_folders(id) ON DELETE SET NULL,
    archived   boolean NOT NULL DEFAULT FALSE,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, board_id)
);
CREATE INDEX board_recipient_state_folder_idx
    ON public.board_recipient_state (user_id, folder_id) WHERE NOT archived;

CREATE TABLE public.board_folder_recipient_state (
    user_id    bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    folder_id  bigint NOT NULL REFERENCES public.board_folders(id) ON DELETE CASCADE,
    parent_id  bigint REFERENCES public.board_folders(id) ON DELETE SET NULL,
    archived   boolean NOT NULL DEFAULT FALSE,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, folder_id)
);
CREATE INDEX board_folder_recipient_state_parent_idx
    ON public.board_folder_recipient_state (user_id, parent_id) WHERE NOT archived;

-- +goose Down
DROP TABLE IF EXISTS public.board_folder_recipient_state;
DROP TABLE IF EXISTS public.board_recipient_state;
DROP TABLE IF EXISTS public.board_folder_company_shares;
DROP TABLE IF EXISTS public.board_folder_user_shares;
DROP TABLE IF EXISTS public.board_company_shares;
DROP TABLE IF EXISTS public.board_user_shares;
DROP TABLE IF EXISTS public.board_shares;
DROP TABLE IF EXISTS public.board_tag_items;
DROP TABLE IF EXISTS public.board_tags;
DROP TABLE IF EXISTS public.boards;
DROP TABLE IF EXISTS public.board_folders;
