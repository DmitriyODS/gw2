-- +goose Up
-- Раздел «Расписание»: регулярная сетка занятий с циклом в 1..4 недели.
--
-- Расписание принадлежит ЧЕЛОВЕКУ (устройство ежедневников, заметок и форм), а
-- коллеги и компании получают его адресно — ТОЛЬКО НА ЧТЕНИЕ: вести занятия
-- может владелец. Расписаний у человека много, и они независимы: своя
-- цикличность, свои категории, свои поля.
--
-- Вхождений на конкретные даты здесь нет и не будет: расписание регулярное,
-- «идёт ли занятие на этой неделе» считается из даты (domain/cycle.go ≡
-- front/src/utils/scheduleCycle.js).

CREATE TABLE public.schedules (
    id          bigserial PRIMARY KEY,
    owner_id    bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    name        varchar(120) NOT NULL,
    -- Длина цикла в неделях: 1 — обычная неделя, 2 — числитель/знаменатель.
    cycle_weeks smallint NOT NULL DEFAULT 1 CHECK (cycle_weeks BETWEEN 1 AND 4),
    -- Понедельник недели №1 цикла: от него нумеруются недели в обе стороны.
    cycle_anchor date NOT NULL,
    -- Названия недель цикла («Числитель», «Знаменатель»); пустая строка на
    -- позиции означает подпись «Неделя N».
    week_labels text[] NOT NULL DEFAULT '{}',
    -- Зона расписания (IANA): по ней считается «сегодня» и «сейчас».
    -- Расписание привязано к месту — учёбе, работе, — а не к устройству.
    timezone    varchar(64) NOT NULL DEFAULT 'Europe/Moscow',
    -- От скольки минут промежуток между занятиями считается окном. Умолчание
    -- расписания: на экране каждый подкручивает порог себе сам.
    gap_min     smallint NOT NULL DEFAULT 20 CHECK (gap_min BETWEEN 5 AND 240),
    position    integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX schedules_owner_idx ON public.schedules (owner_id, position);

-- Категория занятий: она же раскраска блока на шкале и чип-фильтр над ней.
-- Справочник у КАЖДОГО расписания свой.
CREATE TABLE public.schedule_categories (
    id          bigserial PRIMARY KEY,
    schedule_id bigint NOT NULL REFERENCES public.schedules(id) ON DELETE CASCADE,
    name        varchar(80) NOT NULL,
    -- Ключ палитры --tag-* (та же восьмёрка, что у цветов задач).
    color       varchar(16) NOT NULL DEFAULT 'blue',
    position    integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX schedule_categories_schedule_idx
    ON public.schedule_categories (schedule_id, position);

-- Дополнительное поле карточки занятия (преподаватель, аудитория, ссылка на
-- встречу). Набор типов — общее ядро pkg/records БЕЗ файловых: расписание
-- файлов не держит вовсе.
CREATE TABLE public.schedule_fields (
    id            bigserial PRIMARY KEY,
    schedule_id   bigint NOT NULL REFERENCES public.schedules(id) ON DELETE CASCADE,
    label         varchar(120) NOT NULL,
    type          varchar(24) NOT NULL,
    config        jsonb NOT NULL DEFAULT '{}',
    position      integer NOT NULL DEFAULT 0,
    col_span      smallint NOT NULL DEFAULT 1,
    row_span      smallint NOT NULL DEFAULT 1,
    -- Значение видно прямо на блоке шкалы (как «аудитория» в прототипе).
    show_on_block boolean NOT NULL DEFAULT FALSE,
    show_in_card  boolean NOT NULL DEFAULT TRUE,
    created_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX schedule_fields_schedule_idx ON public.schedule_fields (schedule_id, position);

-- Занятие: день недели, время в минутах от полуночи и правило повтора.
CREATE TABLE public.schedule_items (
    id          bigserial PRIMARY KEY,
    schedule_id bigint NOT NULL REFERENCES public.schedules(id) ON DELETE CASCADE,
    -- 0 — понедельник … 6 — воскресенье.
    weekday     smallint NOT NULL CHECK (weekday BETWEEN 0 AND 6),
    start_min   smallint NOT NULL CHECK (start_min BETWEEN 0 AND 1440),
    end_min     smallint NOT NULL CHECK (end_min BETWEEN 0 AND 1440),
    title       varchar(200) NOT NULL,
    -- Короткое имя для узких колонок недели и телефона.
    short       varchar(60) NOT NULL DEFAULT '',
    category_id bigint REFERENCES public.schedule_categories(id) ON DELETE SET NULL,
    -- Номера недель цикла, в которые занятие идёт; пусто — каждую неделю.
    weeks       smallint[] NOT NULL DEFAULT '{}',
    -- Своё правило повтора занятия: «каждые N недель с repeat_from».
    repeat_every smallint,
    repeat_from  date,
    repeat_until date,
    data        jsonb NOT NULL DEFAULT '{}',
    search_text text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT schedule_items_time_check CHECK (end_min > start_min),
    -- Способы повтора взаимоисключающие: либо недели цикла, либо своё правило.
    CONSTRAINT schedule_items_repeat_check CHECK (
        (repeat_every IS NULL AND repeat_from IS NULL AND repeat_until IS NULL)
        OR (repeat_every IS NOT NULL AND repeat_from IS NOT NULL
            AND cardinality(weeks) = 0)
    )
);
CREATE INDEX schedule_items_schedule_idx
    ON public.schedule_items (schedule_id, weekday, start_min);
CREATE INDEX schedule_items_search_idx
    ON public.schedule_items USING gin (search_text gin_trgm_ops);

-- Публичная ссылка (read-only, без авторизации): код в URL — capability.
CREATE TABLE public.schedule_shares (
    id          bigserial PRIMARY KEY,
    schedule_id bigint NOT NULL REFERENCES public.schedules(id) ON DELETE CASCADE,
    code        varchar(64) NOT NULL UNIQUE,
    created_by  bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX schedule_shares_schedule_idx ON public.schedule_shares (schedule_id);

-- Адресный доступ: человеку ЛИБО компании (XOR). Уровень один — чтение.
CREATE TABLE public.schedule_user_shares (
    id          bigserial PRIMARY KEY,
    schedule_id bigint NOT NULL REFERENCES public.schedules(id) ON DELETE CASCADE,
    user_id     bigint REFERENCES public.users(id) ON DELETE CASCADE,
    company_id  bigint REFERENCES public.companies(id) ON DELETE CASCADE,
    created_by  bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT schedule_user_shares_target_check
        CHECK ((user_id IS NOT NULL) <> (company_id IS NOT NULL))
);
-- Адресат один — строка одна: повторная выдача не плодит дубли.
CREATE UNIQUE INDEX schedule_user_shares_user_idx
    ON public.schedule_user_shares (schedule_id, user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX schedule_user_shares_company_idx
    ON public.schedule_user_shares (schedule_id, company_id) WHERE company_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS public.schedule_user_shares;
DROP TABLE IF EXISTS public.schedule_shares;
DROP TABLE IF EXISTS public.schedule_items;
DROP TABLE IF EXISTS public.schedule_fields;
DROP TABLE IF EXISTS public.schedule_categories;
DROP TABLE IF EXISTS public.schedules;
