-- +goose Up
-- Раздел «Календари»: настраиваемые компанией списки записей, привязанных к
-- дате/времени. Календарь имеет набор полей (структуру карточки), которые
-- задаёт администратор, и записи, которые ведут участники компании. Каждая
-- запись обязательно привязана к дате/времени (event_at, без секунд) — по ней
-- она попадает в конкретный день при просмотре по дню/неделе/месяцу. Прочие
-- значения хранятся в JSONB по ключу строкового id поля; search_text —
-- производная строка для сквозного поиска. Поля поддерживают условную
-- видимость (visible_field_id/visible_value): поле показывается в карточке,
-- только когда значение поля-источника совпадает с заданным.

CREATE TABLE public.calendars (
    id          bigserial PRIMARY KEY,
    company_id  bigint NOT NULL REFERENCES public.companies(id) ON DELETE CASCADE,
    name        varchar(120) NOT NULL,
    position    integer NOT NULL DEFAULT 0,
    created_by  bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX calendars_company_idx ON public.calendars (company_id, position);

CREATE TABLE public.calendar_fields (
    id               bigserial PRIMARY KEY,
    calendar_id      bigint NOT NULL REFERENCES public.calendars(id) ON DELETE CASCADE,
    label            varchar(120) NOT NULL,
    type             varchar(32) NOT NULL,
    config           jsonb NOT NULL DEFAULT '{}'::jsonb,
    position         integer NOT NULL DEFAULT 0,
    col_span         integer NOT NULL DEFAULT 1,
    row_span         integer NOT NULL DEFAULT 1,
    show_in_table    boolean NOT NULL DEFAULT true,
    -- Условная видимость: показывать это поле, только когда значение поля
    -- visible_field_id равно visible_value (для checkbox — "true"). NULL — всегда.
    visible_field_id bigint,
    visible_value    text,
    created_at       timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX calendar_fields_calendar_idx ON public.calendar_fields (calendar_id, position);

CREATE TABLE public.calendar_records (
    id          bigserial PRIMARY KEY,
    calendar_id bigint NOT NULL REFERENCES public.calendars(id) ON DELETE CASCADE,
    event_at    timestamptz NOT NULL,
    data        jsonb NOT NULL DEFAULT '{}'::jsonb,
    search_text text NOT NULL DEFAULT '',
    created_by  bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
-- Основной индекс — выборка записей календаря за диапазон дат (день/неделя/месяц).
CREATE INDEX calendar_records_calendar_event_idx ON public.calendar_records (calendar_id, event_at);
-- Триграммный индекс под сквозной ILIKE-поиск по всем текстовым/числовым/датовым
-- полям (search_text пересчитывает сервис при каждой записи). pg_trgm — в baseline.
CREATE INDEX calendar_records_search_idx ON public.calendar_records USING gin (search_text public.gin_trgm_ops);

-- Публичные ссылки на календарь (read-only доступ без авторизации). Код в URL —
-- capability: кто знает код, видит календарь (просмотр/карточки/экспорт), но не
-- может редактировать. Отзыв ссылки = удаление строки.
CREATE TABLE public.calendar_shares (
    id          bigserial PRIMARY KEY,
    calendar_id bigint NOT NULL REFERENCES public.calendars(id) ON DELETE CASCADE,
    code        varchar(40) NOT NULL UNIQUE,
    created_by  bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX calendar_shares_calendar_idx ON public.calendar_shares (calendar_id);

-- +goose Down
DROP TABLE IF EXISTS public.calendar_shares;
DROP TABLE IF EXISTS public.calendar_records;
DROP TABLE IF EXISTS public.calendar_fields;
DROP TABLE IF EXISTS public.calendars;
