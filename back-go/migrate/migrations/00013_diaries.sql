-- +goose Up
-- Раздел «Ежедневник»: личные списки записей-задач пользователя, привязанных к
-- дню. В отличие от компанийного календаря, ежедневник принадлежит ОДНОМУ
-- пользователю (owner_id), не зависит от компании (кросс-компанийный) и имеет
-- ФИКСИРОВАННЫЙ набор полей карточки (день, опц. время начала/конца, название,
-- описание, отметка «выполнено» → архив, связь с задачей tasksvc). Другим
-- ежедневник доступен только read-only через шаринг: публичной ссылкой
-- (diary_shares, код-capability) или адресно (diary_user_shares).

CREATE TABLE public.diaries (
    id          bigserial PRIMARY KEY,
    owner_id    bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    name        varchar(120) NOT NULL,
    position    integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX diaries_owner_idx ON public.diaries (owner_id, position);

CREATE TABLE public.diary_records (
    id             bigserial PRIMARY KEY,
    diary_id       bigint NOT NULL REFERENCES public.diaries(id) ON DELETE CASCADE,
    -- День записи (без времени) — по нему запись попадает в день при просмотре
    -- по дню/неделе/месяцу. Время начала/конца — опциональные минуты от полуночи
    -- (NULL — без времени, «весь день»).
    entry_date     date NOT NULL,
    start_min      integer,
    end_min        integer,
    title          text NOT NULL,
    description    text NOT NULL DEFAULT '',
    -- Выполнена → уходит во вкладку «Архив» (активная вкладка показывает done=false).
    done           boolean NOT NULL DEFAULT false,
    -- Связанная задача tasksvc (создаётся кнопкой в карточке; opaque id, без FK —
    -- задачи в другом сервисе/БД-домене). NULL — нет связи.
    linked_task_id bigint,
    search_text    text NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);
-- Основной индекс — выборка активных записей за диапазон дат по дню/неделе/месяцу.
CREATE INDEX diary_records_diary_date_idx ON public.diary_records (diary_id, done, entry_date);
-- Триграммный индекс под сквозной ILIKE-поиск (search_text = название+описание).
CREATE INDEX diary_records_search_idx ON public.diary_records USING gin (search_text public.gin_trgm_ops);

-- Публичные ссылки (read-only без авторизации). Код в URL — capability.
CREATE TABLE public.diary_shares (
    id          bigserial PRIMARY KEY,
    diary_id    bigint NOT NULL REFERENCES public.diaries(id) ON DELETE CASCADE,
    code        varchar(40) NOT NULL UNIQUE,
    created_by  bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX diary_shares_diary_idx ON public.diary_shares (diary_id);

-- Адресный доступ: ежедневник открыт конкретному пользователю (read-only). Он
-- видит его во вкладке «Поделились». Уникальность пары — один доступ на адресата.
CREATE TABLE public.diary_user_shares (
    id          bigserial PRIMARY KEY,
    diary_id    bigint NOT NULL REFERENCES public.diaries(id) ON DELETE CASCADE,
    user_id     bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (diary_id, user_id)
);
CREATE INDEX diary_user_shares_user_idx ON public.diary_user_shares (user_id);

-- +goose Down
DROP TABLE IF EXISTS public.diary_user_shares;
DROP TABLE IF EXISTS public.diary_shares;
DROP TABLE IF EXISTS public.diary_records;
DROP TABLE IF EXISTS public.diaries;
