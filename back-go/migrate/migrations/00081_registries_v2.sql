-- +goose Up
-- Реестры 2.0: раздел переустроен с компанийного владения на ЛИЧНОЕ с шарингом
-- (устройство заметок и диска). Реестр принадлежит человеку, а компания и
-- коллеги получают к нему доступ адресно. Прежняя видимость сохранена:
-- каждый существующий реестр раздаётся своей компании (см. бэкфилл ниже).
--
-- Здесь же: режим «Учётный реестр» с историей выдач (заменил поле «Наличие»),
-- уровни доступа ссылок с журналом посещений и чанковые загрузки больших
-- файлов.

/* ── Реестр: владелец, учётный режим, подразделы ──────────────────────── */

ALTER TABLE public.registries
    ADD COLUMN owner_id bigint REFERENCES public.users(id) ON DELETE CASCADE,
    -- Учётный реестр: у записей появляется выдача/возврат с историей.
    ADD COLUMN accounting boolean NOT NULL DEFAULT FALSE;

-- Реестр можно вести и без компании (человек не обязан состоять ни в одной).
ALTER TABLE public.registries ALTER COLUMN company_id DROP NOT NULL;

-- Поле-источник подразделов: его варианты становятся вкладками над таблицей.
-- Прежнее имя (tag_field_id, чипы-теги) описывало ту же связь.
ALTER TABLE public.registries RENAME COLUMN tag_field_id TO section_field_id;

/* Ссылка реестра на своё поле и ссылка поля на реестр образуют ЦИКЛ, поэтому
   вставить их в «правильном» порядке нельзя ни при каком: восстановление
   бэкапа льёт таблицы одной транзакцией и упиралось в этот FK. Откладываем
   проверку до коммита — к нему на месте и реестры, и поля. */
ALTER TABLE public.registries DROP CONSTRAINT IF EXISTS registries_tag_field_id_fkey;
ALTER TABLE public.registries
    ADD CONSTRAINT registries_section_field_id_fkey
    FOREIGN KEY (section_field_id) REFERENCES public.registry_fields(id)
    ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED;

-- Владелец: автор реестра, иначе создатель компании, иначе старший её
-- участник, иначе любой аккаунт — реестр не должен остаться без хозяина.
UPDATE public.registries r SET owner_id = COALESCE(
    r.created_by,
    (SELECT c.created_by FROM public.companies c WHERE c.id = r.company_id),
    (SELECT uc.user_id FROM public.user_companies uc
        JOIN public.roles ro ON ro.id = uc.role_id
       WHERE uc.company_id = r.company_id
       ORDER BY ro.level DESC, uc.user_id LIMIT 1),
    (SELECT u.id FROM public.users u ORDER BY u.is_super_admin DESC, u.id LIMIT 1)
) WHERE r.owner_id IS NULL;

-- Владельца не нашлось только если в базе нет ни одного пользователя — тогда
-- реестр и показать некому.
DELETE FROM public.registries WHERE owner_id IS NULL;
ALTER TABLE public.registries ALTER COLUMN owner_id SET NOT NULL;
CREATE INDEX registries_owner_idx ON public.registries (owner_id, position);

/* ── Адресный доступ: человеку или всей компании ──────────────────────── */

-- Устройство как у диска: одна таблица, адресат — либо пользователь, либо
-- компания. access: view — смотреть и выгружать, edit — плюс вести записи,
-- admin — плюс менять структуру реестра.
CREATE TABLE public.registry_user_shares (
    id          bigserial PRIMARY KEY,
    registry_id bigint NOT NULL REFERENCES public.registries(id) ON DELETE CASCADE,
    user_id     bigint REFERENCES public.users(id) ON DELETE CASCADE,
    company_id  bigint REFERENCES public.companies(id) ON DELETE CASCADE,
    access      text NOT NULL DEFAULT 'view' CHECK (access IN ('view', 'edit', 'admin')),
    created_by  bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_registry_share_who CHECK (
        (user_id IS NOT NULL AND company_id IS NULL)
        OR (user_id IS NULL AND company_id IS NOT NULL)
    )
);
CREATE INDEX registry_user_shares_registry_idx ON public.registry_user_shares (registry_id);
-- Один адресат — одна строка: повторная выдача обновляет уровень доступа.
CREATE UNIQUE INDEX registry_user_shares_user_idx
    ON public.registry_user_shares (registry_id, user_id) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX registry_user_shares_company_idx
    ON public.registry_user_shares (registry_id, company_id) WHERE company_id IS NOT NULL;

-- Прежняя видимость: реестр был виден всей своей компании и любой участник
-- вёл записи — это ровно уровень edit для компании.
INSERT INTO public.registry_user_shares (registry_id, company_id, access, created_by)
SELECT r.id, r.company_id, 'edit', r.owner_id
  FROM public.registries r
 WHERE r.company_id IS NOT NULL;

-- Структуру реестра правил администратор компании — сохраняем это поимённо,
-- иначе после переезда её сможет менять только владелец.
INSERT INTO public.registry_user_shares (registry_id, user_id, access, created_by)
SELECT DISTINCT r.id, uc.user_id, 'admin', r.owner_id
  FROM public.registries r
  JOIN public.user_companies uc ON uc.company_id = r.company_id
  JOIN public.roles ro ON ro.id = uc.role_id AND ro.level >= 3
 WHERE r.company_id IS NOT NULL AND uc.user_id <> r.owner_id;

/* ── Внешние ссылки: уровень доступа, вход по паролю, журнал посещений ── */

ALTER TABLE public.registry_shares
    DROP CONSTRAINT IF EXISTS registry_shares_access_check;
ALTER TABLE public.registry_shares
    ADD CONSTRAINT registry_shares_access_check CHECK (access IN ('view', 'edit', 'admin'));

ALTER TABLE public.registry_shares
    -- Ссылка «только для своих»: до открытия реестра гость должен войти или
    -- завести аккаунт, поэтому в журнале у перехода есть имя, а не только адрес.
    ADD COLUMN require_auth boolean NOT NULL DEFAULT FALSE,
    -- Своё название ссылки: их у реестра много, и отзывать нужную удобнее по
    -- слову («для подрядчика»), а не по коду.
    ADD COLUMN name varchar(120) NOT NULL DEFAULT '';

-- Журнал переходов: кто, когда и откуда открывал ссылку. Гость аккаунта не
-- имеет — у него остаётся только адрес и время.
CREATE TABLE public.registry_share_visits (
    id         bigserial PRIMARY KEY,
    share_id   bigint NOT NULL REFERENCES public.registry_shares(id) ON DELETE CASCADE,
    user_id    bigint REFERENCES public.users(id) ON DELETE SET NULL,
    ip         varchar(64) NOT NULL DEFAULT '',
    user_agent varchar(400) NOT NULL DEFAULT '',
    visited_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX registry_share_visits_share_idx
    ON public.registry_share_visits (share_id, visited_at DESC);

/* ── Учётный реестр: выдачи, продления, возвраты ──────────────────────── */

-- Выдача позиции. Открытая (returned_at IS NULL) у записи одна — её состояние
-- и показывает таблица плашкой «выдано до / просрочено».
CREATE TABLE public.registry_issues (
    id           bigserial PRIMARY KEY,
    registry_id  bigint NOT NULL REFERENCES public.registries(id) ON DELETE CASCADE,
    record_id    bigint NOT NULL REFERENCES public.registry_records(id) ON DELETE CASCADE,
    -- Получатель: свободное ФИО (его может не быть в системе вовсе) и
    -- необязательная ссылка на аккаунт.
    holder_name    varchar(200) NOT NULL DEFAULT '',
    holder_phone   varchar(40) NOT NULL DEFAULT '',
    holder_user_id bigint REFERENCES public.users(id) ON DELETE SET NULL,
    issued_by      bigint REFERENCES public.users(id) ON DELETE SET NULL,
    issued_at      timestamptz NOT NULL DEFAULT now(),
    due_at         timestamptz,
    returned_at    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX registry_issues_record_idx ON public.registry_issues (record_id, issued_at DESC);
-- Одна позиция не может быть выдана дважды одновременно.
CREATE UNIQUE INDEX registry_issues_open_idx
    ON public.registry_issues (record_id) WHERE returned_at IS NULL;

-- История движения: выдача, продление, возврат — каждое со своим сроком и
-- комментарием. Отдельной таблицей, потому что продлений бывает много.
CREATE TABLE public.registry_issue_events (
    id         bigserial PRIMARY KEY,
    issue_id   bigint NOT NULL REFERENCES public.registry_issues(id) ON DELETE CASCADE,
    kind       text NOT NULL CHECK (kind IN ('issue', 'extend', 'return')),
    due_at     timestamptz,
    comment    varchar(1000) NOT NULL DEFAULT '',
    actor_id   bigint REFERENCES public.users(id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX registry_issue_events_issue_idx
    ON public.registry_issue_events (issue_id, created_at);

/* ── Перенос структуры полей на набор 2.0 ─────────────────────────────── */

-- Длинный текст стал самостоятельным типом (был флагом config.multiline).
UPDATE public.registry_fields
   SET type = 'textarea', config = config - 'multiline'
 WHERE type = 'text' AND (config->>'multiline')::boolean IS TRUE;
UPDATE public.registry_fields SET config = config - 'multiline'
 WHERE type = 'text' AND config ? 'multiline';

-- Части даты включаются по одной (были year/month_day/time).
-- +goose StatementBegin
DO $$
DECLARE
    tbl text;
BEGIN
    FOREACH tbl IN ARRAY ARRAY['registry_fields', 'calendar_fields'] LOOP
        EXECUTE format($f$
            UPDATE public.%I SET config = (config - 'month_day' - 'time')
                || jsonb_build_object(
                    'year',    COALESCE((config->>'year')::boolean, TRUE),
                    'month',   COALESCE((config->>'month_day')::boolean, TRUE),
                    'day',     COALESCE((config->>'month_day')::boolean, TRUE),
                    'hours',   COALESCE((config->>'time')::boolean, TRUE),
                    'minutes', COALESCE((config->>'time')::boolean, TRUE),
                    'seconds', FALSE)
             WHERE type = 'datetime' AND config ? 'month_day'
        $f$, tbl);
    END LOOP;
END $$;
-- +goose StatementEnd

-- Карточка реестра делится на четверти вместо третей: поле во всю строку
-- остаётся во всю строку, узкие сохраняют свою ширину.
UPDATE public.registry_fields SET col_span = 4 WHERE col_span >= 3;

/* ── Перенос поля «Наличие» в учётный режим ───────────────────────────── */

-- Реестр, который вёл наличие позиций, становится учётным.
UPDATE public.registries r SET accounting = TRUE
 WHERE EXISTS (SELECT 1 FROM public.registry_fields f
                WHERE f.registry_id = r.id AND f.type = 'stock');

-- Забранные позиции превращаются в открытые выдачи: получателя прежнее поле
-- не знало, поэтому в истории он «не указан», а срок берётся из «забрали до».
-- +goose StatementBegin
DO $$
DECLARE
    rec RECORD;
    new_issue bigint;
BEGIN
    FOR rec IN
        SELECT rr.id AS record_id, rr.registry_id, f.id AS field_id,
               rr.data->f.id::text->>'until' AS until
          FROM public.registry_records rr
          JOIN public.registry_fields f
            ON f.registry_id = rr.registry_id AND f.type = 'stock'
         WHERE (rr.data->f.id::text->>'taken')::boolean IS TRUE
    LOOP
        INSERT INTO public.registry_issues
            (registry_id, record_id, holder_name, due_at, issued_at)
        VALUES (rec.registry_id, rec.record_id, 'Не указан',
                NULLIF(rec.until, '')::timestamptz, now())
        ON CONFLICT DO NOTHING
        RETURNING id INTO new_issue;

        IF new_issue IS NOT NULL THEN
            INSERT INTO public.registry_issue_events (issue_id, kind, due_at, comment)
            VALUES (new_issue, 'issue', NULLIF(rec.until, '')::timestamptz,
                    'Перенесено из поля «Наличие»');
        END IF;
    END LOOP;
END $$;
-- +goose StatementEnd

-- Значения прежнего поля больше не нужны: их место заняла история выдач.
-- Из строки поиска уходит вклад «забрали …», иначе записи продолжали бы
-- находиться по слову, которого в них уже нет.
UPDATE public.registry_records rr
   SET data = rr.data - f.id::text,
       search_text = btrim(regexp_replace(rr.search_text,
           'забрали( \d{4}-\d{2}-\d{2})?', '', 'g'))
  FROM public.registry_fields f
 WHERE f.registry_id = rr.registry_id AND f.type = 'stock'
   AND rr.data ? f.id::text;

DELETE FROM public.registry_fields WHERE type = 'stock';

-- +goose Down
DELETE FROM public.registry_fields WHERE type = 'stock';
DROP TABLE IF EXISTS public.registry_issue_events;
DROP TABLE IF EXISTS public.registry_issues;
DROP TABLE IF EXISTS public.registry_share_visits;
DROP TABLE IF EXISTS public.registry_user_shares;

ALTER TABLE public.registry_shares
    DROP COLUMN IF EXISTS require_auth,
    DROP COLUMN IF EXISTS name;
ALTER TABLE public.registry_shares
    DROP CONSTRAINT IF EXISTS registry_shares_access_check;
UPDATE public.registry_shares SET access = 'view' WHERE access NOT IN ('view', 'edit');
ALTER TABLE public.registry_shares
    ADD CONSTRAINT registry_shares_access_check CHECK (access IN ('view', 'edit'));

DROP INDEX IF EXISTS public.registries_owner_idx;
ALTER TABLE public.registries RENAME COLUMN section_field_id TO tag_field_id;
ALTER TABLE public.registries
    DROP COLUMN IF EXISTS accounting,
    DROP COLUMN IF EXISTS owner_id;
DELETE FROM public.registries WHERE company_id IS NULL;
ALTER TABLE public.registries ALTER COLUMN company_id SET NOT NULL;
