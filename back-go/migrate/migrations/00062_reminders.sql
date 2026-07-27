-- +goose Up
-- Раздел «Напоминания»: личные напоминания пользователя на дату и время —
-- разовые либо повторяющиеся (ежедневно / по рабочим дням / по выбранным дням
-- недели / ежемесячно / ежегодно), со свободным текстом либо привязкой к записи
-- ежедневника или календаря. Напоминание принадлежит ОДНОМУ пользователю
-- (owner_id) и не зависит от компании (кросс-компанийное, как ежедневник).
--
-- Времена — UTC; timezone (IANA) нужен повторам: «каждый день в 9:00» обязано
-- оставаться девятью утра пользователя и после перевода часов. active — ждёт
-- своего часа; сработавшее разовое становится неактивным (журнал), у повтора
-- remind_at переезжает на следующий срок. fired_seq растёт при каждом заборе
-- планировщиком — им же ClaimDue гарантирует «одно срабатывание — одна
-- доставка» при нескольких инстансах сервиса.
CREATE TABLE public.reminders (
    id              bigserial PRIMARY KEY,
    owner_id        bigint NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    title           varchar(300) NOT NULL,
    note            text NOT NULL DEFAULT '',
    remind_at       timestamptz NOT NULL,
    timezone        varchar(64) NOT NULL DEFAULT 'UTC',

    repeat_kind     varchar(16) NOT NULL DEFAULT 'none'
        CHECK (repeat_kind IN ('none', 'daily', 'weekdays', 'weekly', 'monthly', 'yearly')),
    repeat_interval integer NOT NULL DEFAULT 1,
    repeat_days     integer[] NOT NULL DEFAULT '{}',
    repeat_until    timestamptz,

    link_kind       varchar(16) NOT NULL DEFAULT 'none'
        CHECK (link_kind IN ('none', 'diary', 'calendar')),
    link_parent_id  bigint NOT NULL DEFAULT 0,
    link_record_id  bigint NOT NULL DEFAULT 0,
    link_title      varchar(300) NOT NULL DEFAULT '',
    link_lead_min   integer NOT NULL DEFAULT 0,

    active          boolean NOT NULL DEFAULT TRUE,
    fired_seq       integer NOT NULL DEFAULT 0,
    last_fired_at   timestamptz,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

-- Список раздела и живая плитка: свои напоминания по возрастанию срока.
CREATE INDEX reminders_owner_idx ON public.reminders (owner_id, active, remind_at);
-- Горячий путь планировщика: наступившие сроки по всей платформе.
CREATE INDEX reminders_due_idx ON public.reminders (remind_at) WHERE active;
-- Привязки: раздел, правящий запись, обновляет снимок в её напоминаниях.
CREATE INDEX reminders_link_idx ON public.reminders (owner_id, link_kind, link_record_id)
    WHERE link_kind <> 'none';

-- +goose Down
DROP TABLE IF EXISTS public.reminders;
