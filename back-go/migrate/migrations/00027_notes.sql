-- +goose Up
-- Раздел «Заметки»: личные заметки пользователя с rich-текстом (документ
-- TipTap в JSONB) и группами. Заметка принадлежит ОДНОМУ пользователю
-- (owner_id), не зависит от компании (кросс-компанийная, как ежедневник);
-- другим доступна только по публичной ссылке (note_shares, код-capability) в
-- режиме «чтение» или «чтение и редактирование». text_content — плоский текст,
-- пересчитывается сервером из doc при сохранении (поиск и txt-экспорт).

CREATE TABLE public.notes (
    id           bigserial PRIMARY KEY,
    owner_id     bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    title        varchar(300) NOT NULL DEFAULT '',
    doc          jsonb NOT NULL DEFAULT '{}',
    text_content text NOT NULL DEFAULT '',
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX notes_owner_idx ON public.notes (owner_id, updated_at DESC);
-- Триграммный индекс под сквозной ILIKE-поиск по заголовку и тексту.
CREATE INDEX notes_search_idx ON public.notes
    USING gin ((title || ' ' || text_content) public.gin_trgm_ops);

-- Группы заметок (личные папки-фильтры владельца). Заметка может входить в
-- несколько групп, одну или ни одной — связи в note_group_items.
CREATE TABLE public.note_groups (
    id          bigserial PRIMARY KEY,
    owner_id    bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    name        varchar(100) NOT NULL,
    position    integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX note_groups_owner_idx ON public.note_groups (owner_id, position);

CREATE TABLE public.note_group_items (
    note_id   bigint NOT NULL REFERENCES public.notes(id) ON DELETE CASCADE,
    group_id  bigint NOT NULL REFERENCES public.note_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, group_id)
);
CREATE INDEX note_group_items_group_idx ON public.note_group_items (group_id);

-- Публичные ссылки на заметку (без авторизации). Код в URL — capability;
-- access задаёт режим: view — только чтение, edit — чтение и редактирование.
CREATE TABLE public.note_shares (
    id          bigserial PRIMARY KEY,
    note_id     bigint NOT NULL REFERENCES public.notes(id) ON DELETE CASCADE,
    code        varchar(40) NOT NULL UNIQUE,
    access      varchar(8) NOT NULL CHECK (access IN ('view', 'edit')),
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX note_shares_note_idx ON public.note_shares (note_id);

-- +goose Down
DROP TABLE IF EXISTS public.note_shares;
DROP TABLE IF EXISTS public.note_group_items;
DROP TABLE IF EXISTS public.note_groups;
DROP TABLE IF EXISTS public.notes;
