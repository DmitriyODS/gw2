-- +goose Up
-- Раздел «Формы и опросы»: конструктор форм по образцу Google Forms.
--
-- Форма принадлежит ЧЕЛОВЕКУ (устройство реестров, заметок и диска), а коллеги
-- и компании получают её адресно. Уровней доступа три и они вложены:
--   respond — только заполнить (это же и «назначение»: у адресата появляется
--             обязанность ответить и, при желании автора, срок);
--   view    — плюс видеть ответы и сводку;
--   edit    — плюс менять саму форму.
-- Владелец сильнее любого уровня и один может форму удалить.

/* ── Форма ───────────────────────────────────────────────────────────── */

CREATE TABLE public.forms (
    id          bigserial PRIMARY KEY,
    owner_id    bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    -- Компания, в которой форма заведена: определяет, чья квота платит за
    -- файлы ответов, и что предлагать в «поделиться с компанией».
    company_id  bigint REFERENCES public.companies(id) ON DELETE SET NULL,
    title       varchar(200) NOT NULL,
    description text NOT NULL DEFAULT '',
    -- draft — черновик (ответы не принимаются, ссылка не работает),
    -- open — приём открыт, closed — приём закрыт вручную.
    status      text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'open', 'closed')),

    -- Приём ответов.
    -- allow_anonymous — отвечать можно и без входа в аккаунт (ссылку открыл
    -- посторонний). Выключено — по ссылке пустят только вошедшего.
    allow_anonymous  boolean NOT NULL DEFAULT TRUE,
    -- one_response — один ответ от человека (для вошедших; аноним не опознаётся).
    one_response     boolean NOT NULL DEFAULT FALSE,
    -- allow_edit — разрешить менять свой уже отправленный ответ.
    allow_edit       boolean NOT NULL DEFAULT FALSE,
    -- collect_email — спрашивать почту отвечающего отдельным полем.
    collect_email    boolean NOT NULL DEFAULT FALSE,
    show_progress    boolean NOT NULL DEFAULT TRUE,
    shuffle_questions boolean NOT NULL DEFAULT FALSE,
    -- Текст после отправки («Спасибо! Ответ записан»).
    confirmation     varchar(2000) NOT NULL DEFAULT '',
    -- Показывать ли отвечающему сводку ответов после отправки.
    show_summary     boolean NOT NULL DEFAULT FALSE,

    -- Режим теста: у вопросов появляются баллы и правильные ответы.
    quiz             boolean NOT NULL DEFAULT FALSE,
    -- Когда отвечающий видит оценку: сразу после отправки либо после того, как
    -- автор проверит ответы вручную.
    quiz_release     text NOT NULL DEFAULT 'immediately'
                     CHECK (quiz_release IN ('immediately', 'manual')),
    -- Показывать ли правильные ответы и пояснения вместе с оценкой.
    quiz_show_answers boolean NOT NULL DEFAULT TRUE,

    -- Окно приёма: до opens_at форма ещё не открыта, после closes_at закрыта.
    opens_at   timestamptz,
    closes_at  timestamptz,
    -- Потолок числа ответов (0 — без ограничения).
    max_responses integer NOT NULL DEFAULT 0,

    position   integer NOT NULL DEFAULT 0,
    created_by bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX forms_owner_idx ON public.forms (owner_id, position);

/* ── Разделы-страницы и вопросы ───────────────────────────────────────── */

-- Раздел формы: заголовок, описание и переход после него. У формы всегда есть
-- хотя бы один раздел — на него и ложатся вопросы «первой страницы».
CREATE TABLE public.form_sections (
    id          bigserial PRIMARY KEY,
    form_id     bigint NOT NULL REFERENCES public.forms(id) ON DELETE CASCADE,
    title       varchar(200) NOT NULL DEFAULT '',
    description varchar(2000) NOT NULL DEFAULT '',
    position    integer NOT NULL DEFAULT 0,
    -- Куда идти после раздела: next — следующий по порядку, section — к
    -- next_section_id, submit — отправить форму.
    next_action text NOT NULL DEFAULT 'next' CHECK (next_action IN ('next', 'section', 'submit')),
    -- Ссылка раздела на раздел ТОЙ ЖЕ таблицы: восстановление бэкапа льёт её
    -- одной транзакцией и в «правильном» порядке вставить не может — проверку
    -- откладываем до коммита.
    next_section_id bigint REFERENCES public.form_sections(id)
                    ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX form_sections_form_idx ON public.form_sections (form_id, position);

-- Вопрос. config хранит настройки конкретного типа (варианты, строки/столбцы
-- сетки, границы шкалы, проверку текста, потолок файлов, переходы по вариантам),
-- answer_key — правильный ответ и пояснение режима теста.
CREATE TABLE public.form_questions (
    id          bigserial PRIMARY KEY,
    form_id     bigint NOT NULL REFERENCES public.forms(id) ON DELETE CASCADE,
    section_id  bigint NOT NULL REFERENCES public.form_sections(id) ON DELETE CASCADE,
    type        varchar(32) NOT NULL,
    title       varchar(500) NOT NULL DEFAULT '',
    description varchar(2000) NOT NULL DEFAULT '',
    required    boolean NOT NULL DEFAULT FALSE,
    config      jsonb NOT NULL DEFAULT '{}'::jsonb,
    position    integer NOT NULL DEFAULT 0,
    points      integer NOT NULL DEFAULT 0,
    answer_key  jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX form_questions_form_idx ON public.form_questions (form_id, position);
CREATE INDEX form_questions_section_idx ON public.form_questions (section_id, position);

/* ── Ответы ──────────────────────────────────────────────────────────── */

-- Одна отправка формы. answers — карта строкового id вопроса в значение (форма
-- значения зависит от типа), search_text — производная строка для поиска по
-- ответам.
CREATE TABLE public.form_responses (
    id          bigserial PRIMARY KEY,
    form_id     bigint NOT NULL REFERENCES public.forms(id) ON DELETE CASCADE,
    -- NULL — отвечал гость (аноним): аккаунта у него нет.
    user_id     bigint REFERENCES public.users(id) ON DELETE SET NULL,
    email       varchar(200) NOT NULL DEFAULT '',
    -- Имя, которым представился гость (у вошедшего берётся из аккаунта).
    name        varchar(200) NOT NULL DEFAULT '',
    answers     jsonb NOT NULL DEFAULT '{}'::jsonb,
    search_text text NOT NULL DEFAULT '',
    -- Режим теста: набранный и максимальный балл, признак проверки.
    score       integer NOT NULL DEFAULT 0,
    max_score   integer NOT NULL DEFAULT 0,
    graded      boolean NOT NULL DEFAULT FALSE,
    -- Через какую внешнюю ссылку пришёл ответ (NULL — из самого приложения).
    -- Внешний ключ добавляется ниже: таблица ссылок создаётся после ответов.
    share_id    bigint,
    /* single — копия настройки «один ответ от человека» на момент отправки.
       Денормализована сюда ради частичного уникального индекса ниже: гонку двух
       вкладок обязана отбивать база, а условие индекса не умеет заглядывать в
       соседнюю таблицу. */
    single      boolean NOT NULL DEFAULT FALSE,
    ip          varchar(64) NOT NULL DEFAULT '',
    user_agent  varchar(400) NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX form_responses_form_idx ON public.form_responses (form_id, created_at DESC);
CREATE INDEX form_responses_user_idx ON public.form_responses (user_id, form_id);
CREATE UNIQUE INDEX form_responses_single_idx
    ON public.form_responses (form_id, user_id) WHERE user_id IS NOT NULL AND single;
CREATE INDEX form_responses_search_idx
    ON public.form_responses USING gin (search_text public.gin_trgm_ops);

/* ── Доступ: человеку или всей компании ──────────────────────────────── */

-- Устройство как у реестров и диска: одна таблица, адресат — либо пользователь,
-- либо компания. Уровень respond и есть «назначение»: адресат обязан ответить,
-- у него форма появляется во вкладке «Мне назначены», и его считает контроль
-- исполнения.
CREATE TABLE public.form_user_shares (
    id         bigserial PRIMARY KEY,
    form_id    bigint NOT NULL REFERENCES public.forms(id) ON DELETE CASCADE,
    user_id    bigint REFERENCES public.users(id) ON DELETE CASCADE,
    company_id bigint REFERENCES public.companies(id) ON DELETE CASCADE,
    access     text NOT NULL DEFAULT 'respond' CHECK (access IN ('respond', 'view', 'edit')),
    -- Срок ответа назначенным: по нему уходит одно напоминание тем, кто ещё не
    -- ответил (планировщик внутри formsvc).
    due_at     timestamptz,
    reminded_at timestamptz,
    created_by bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_form_share_who CHECK (
        (user_id IS NOT NULL AND company_id IS NULL)
        OR (user_id IS NULL AND company_id IS NOT NULL)
    )
);
CREATE INDEX form_user_shares_form_idx ON public.form_user_shares (form_id);
-- Один адресат — одна строка: повторная выдача меняет уровень и срок.
CREATE UNIQUE INDEX form_user_shares_user_idx
    ON public.form_user_shares (form_id, user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX form_user_shares_company_idx
    ON public.form_user_shares (form_id, company_id) WHERE company_id IS NOT NULL;
-- Планировщику напоминаний: несработавшие сроки.
CREATE INDEX form_user_shares_due_idx
    ON public.form_user_shares (due_at) WHERE due_at IS NOT NULL AND reminded_at IS NULL;

/* ── Внешние ссылки и журнал переходов ───────────────────────────────── */

-- Ссылка на заполнение: код в адресе — capability. Требование входа — свойство
-- САМОЙ ссылки: одну форму раздают и «своим», и наружу.
CREATE TABLE public.form_shares (
    id           bigserial PRIMARY KEY,
    form_id      bigint NOT NULL REFERENCES public.forms(id) ON DELETE CASCADE,
    code         varchar(40) NOT NULL UNIQUE,
    name         varchar(120) NOT NULL DEFAULT '',
    require_auth boolean NOT NULL DEFAULT FALSE,
    created_by   bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX form_shares_form_idx ON public.form_shares (form_id);

-- Ссылка в form_responses.share_id объявлена выше, до создания form_shares:
-- добавляем внешний ключ теперь, когда обе таблицы на месте.
ALTER TABLE public.form_responses
    ADD CONSTRAINT form_responses_share_id_fkey
    FOREIGN KEY (share_id) REFERENCES public.form_shares(id) ON DELETE SET NULL;

CREATE TABLE public.form_share_visits (
    id         bigserial PRIMARY KEY,
    share_id   bigint NOT NULL REFERENCES public.form_shares(id) ON DELETE CASCADE,
    user_id    bigint REFERENCES public.users(id) ON DELETE SET NULL,
    ip         varchar(64) NOT NULL DEFAULT '',
    user_agent varchar(400) NOT NULL DEFAULT '',
    visited_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX form_share_visits_share_idx
    ON public.form_share_visits (share_id, visited_at DESC);

-- +goose Down
DROP TABLE IF EXISTS public.form_share_visits;
DROP TABLE IF EXISTS public.form_responses;
DROP TABLE IF EXISTS public.form_shares;
DROP TABLE IF EXISTS public.form_user_shares;
DROP TABLE IF EXISTS public.form_questions;
DROP TABLE IF EXISTS public.form_sections;
DROP TABLE IF EXISTS public.forms;
