-- +goose Up
-- Раздел «Реестры»: настраиваемые таблицы-справочники компаний. Реестр имеет
-- набор полей (структуру карточки), которые задаёт администратор, и записи,
-- которые ведут участники компании. Значения записей хранятся в JSONB по ключу
-- строкового id поля; search_text — производная строка для сквозного поиска.

CREATE TABLE public.registries (
    id          bigserial PRIMARY KEY,
    company_id  bigint NOT NULL REFERENCES public.companies(id) ON DELETE CASCADE,
    name        varchar(120) NOT NULL,
    position    integer NOT NULL DEFAULT 0,
    created_by  bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX registries_company_idx ON public.registries (company_id, position);

CREATE TABLE public.registry_fields (
    id            bigserial PRIMARY KEY,
    registry_id   bigint NOT NULL REFERENCES public.registries(id) ON DELETE CASCADE,
    label         varchar(120) NOT NULL,
    type          varchar(32) NOT NULL,
    config        jsonb NOT NULL DEFAULT '{}'::jsonb,
    position      integer NOT NULL DEFAULT 0,
    col_span      integer NOT NULL DEFAULT 1,
    row_span      integer NOT NULL DEFAULT 1,
    show_in_table boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX registry_fields_registry_idx ON public.registry_fields (registry_id, position);

CREATE TABLE public.registry_records (
    id          bigserial PRIMARY KEY,
    registry_id bigint NOT NULL REFERENCES public.registries(id) ON DELETE CASCADE,
    data        jsonb NOT NULL DEFAULT '{}'::jsonb,
    search_text text NOT NULL DEFAULT '',
    created_by  bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX registry_records_registry_idx ON public.registry_records (registry_id, created_at DESC);
-- Триграммный индекс под сквозной ILIKE-поиск по всем текстовым/числовым/датовым
-- полям (search_text пересчитывает сервис при каждой записи). pg_trgm — в baseline.
CREATE INDEX registry_records_search_idx ON public.registry_records USING gin (search_text public.gin_trgm_ops);

-- +goose Down
DROP TABLE IF EXISTS public.registry_records;
DROP TABLE IF EXISTS public.registry_fields;
DROP TABLE IF EXISTS public.registries;
