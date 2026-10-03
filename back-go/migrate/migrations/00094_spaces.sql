-- +goose Up
-- Пространства «Моё» и команды. Вещь инструмента лежит ровно в одном
-- пространстве — колонка company_id: NULL означает личное пространство
-- owner_id, иначе вещь принадлежит команде, а owner_id — её автор. Рядовые
-- участники команды получают уровень team_access; автор и администраторы
-- команды распоряжаются вещью целиком. Удаление команды не уносит её вещи:
-- они уезжают в личное пространство автора (ON DELETE SET NULL).

-- Файлы, сменившие плательщика при переезде: журнал биллинга переписывается
-- в конце миграции. Без этого сверка «Хранилища» стёрла бы их у прежнего
-- плательщика как сирот.
CREATE TEMP TABLE space_moves (
    storage_key text PRIMARY KEY,
    user_id     bigint NOT NULL,
    company_id  bigint
) ON COMMIT DROP;

-- ── Календари ───────────────────────────────────────────────────
-- Раньше календарь принадлежал компании целиком. Теперь у него есть автор —
-- тот, кто завёл, а без него (аккаунт удалён) — создатель компании.
ALTER TABLE public.calendars
    ADD COLUMN owner_id bigint REFERENCES public.users(id) ON DELETE CASCADE,
    ADD COLUMN team_access text NOT NULL DEFAULT 'edit'
        CHECK (team_access IN ('view', 'edit', 'admin'));
UPDATE public.calendars cal
   SET owner_id = COALESCE(cal.created_by, c.created_by,
                           (SELECT min(uc.user_id) FROM public.user_companies uc
                             WHERE uc.company_id = cal.company_id))
  FROM public.companies c
 WHERE c.id = cal.company_id;
-- Календарь компании, от которой не осталось ни одного человека, вести некому.
DELETE FROM public.calendars WHERE owner_id IS NULL;
ALTER TABLE public.calendars ALTER COLUMN owner_id SET NOT NULL,
    ALTER COLUMN company_id DROP NOT NULL,
    DROP CONSTRAINT calendars_company_id_fkey,
    ADD CONSTRAINT calendars_company_id_fkey FOREIGN KEY (company_id)
        REFERENCES public.companies(id) ON DELETE SET NULL;
CREATE INDEX calendars_owner_idx ON public.calendars (owner_id) WHERE company_id IS NULL;

-- ── Реестры ─────────────────────────────────────────────────────
ALTER TABLE public.registries
    ADD COLUMN team_access text NOT NULL DEFAULT 'edit'
        CHECK (team_access IN ('view', 'edit', 'admin'));

-- Права не расширяются молча: реестр остаётся командным, только если его
-- раздали своей же компании (уровень шары становится уровнем участников),
-- иначе он личный — каким его и видели до сих пор.
UPDATE public.registries reg SET team_access = sh.access
  FROM public.registry_user_shares sh
 WHERE sh.registry_id = reg.id AND sh.company_id = reg.company_id;

-- Личный реестр платит хозяин, а не создатель компании.
INSERT INTO space_moves (storage_key, user_id, company_id)
SELECT DISTINCT f.key, reg.owner_id, NULL::bigint
  FROM public.registries reg
  JOIN public.registry_records rec ON rec.registry_id = reg.id
  CROSS JOIN LATERAL jsonb_each(rec.data) d(fid, val)
  CROSS JOIN LATERAL (VALUES (d.val ->> 'path'), (d.val ->> 'thumb')) f(key)
 WHERE reg.company_id IS NOT NULL
   AND jsonb_typeof(d.val) = 'object' AND f.key IS NOT NULL AND f.key <> ''
   AND NOT EXISTS (SELECT 1 FROM public.registry_user_shares sh
                    WHERE sh.registry_id = reg.id AND sh.company_id = reg.company_id)
ON CONFLICT (storage_key) DO NOTHING;

UPDATE public.registries reg SET company_id = NULL
 WHERE company_id IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM public.registry_user_shares sh
                    WHERE sh.registry_id = reg.id AND sh.company_id = reg.company_id);
DELETE FROM public.registry_user_shares sh USING public.registries reg
 WHERE sh.registry_id = reg.id AND sh.company_id = reg.company_id;

ALTER TABLE public.registries DROP CONSTRAINT registries_company_id_fkey,
    ADD CONSTRAINT registries_company_id_fkey FOREIGN KEY (company_id)
        REFERENCES public.companies(id) ON DELETE SET NULL;

-- ── Формы ───────────────────────────────────────────────────────
-- Форма остаётся командной, только если её назначили или раздали своей же
-- компании; уровень участников — уровень этой шары. Саму шару не трогаем: на
-- ней держится назначение (срок ответа, напоминание, «кто ответил»).
-- Умолчание новой формы команды — «заполнить»: с просмотром участники видели
-- бы чужие ответы.
ALTER TABLE public.forms
    ADD COLUMN team_access text NOT NULL DEFAULT 'respond'
        CHECK (team_access IN ('respond', 'view', 'edit'));
UPDATE public.forms f
   SET team_access = (SELECT CASE max(CASE sh.access WHEN 'edit' THEN 3 WHEN 'view' THEN 2 ELSE 1 END)
                               WHEN 3 THEN 'edit' WHEN 2 THEN 'view' ELSE 'respond' END
                        FROM public.form_user_shares sh
                       WHERE sh.form_id = f.id AND sh.company_id = f.company_id)
 WHERE EXISTS (SELECT 1 FROM public.form_user_shares sh
                WHERE sh.form_id = f.id AND sh.company_id = f.company_id);

-- Личную форму оплачивает её хозяин, а не создатель компании.
INSERT INTO space_moves (storage_key, user_id, company_id)
SELECT DISTINCT k.key, f.owner_id, NULL::bigint
  FROM public.forms f
  JOIN public.form_responses fr ON fr.form_id = f.id
  CROSS JOIN LATERAL jsonb_each(fr.answers) a(qid, val)
  CROSS JOIN LATERAL jsonb_array_elements(CASE WHEN jsonb_typeof(a.val) = 'array'
                                               THEN a.val ELSE '[]'::jsonb END) item
  CROSS JOIN LATERAL (VALUES (item ->> 'path'), (item ->> 'thumb')) k(key)
 WHERE f.company_id IS NOT NULL
   AND jsonb_typeof(item) = 'object' AND k.key IS NOT NULL AND k.key <> ''
   AND NOT EXISTS (SELECT 1 FROM public.form_user_shares sh
                    WHERE sh.form_id = f.id AND sh.company_id = f.company_id)
ON CONFLICT (storage_key) DO NOTHING;

UPDATE public.forms f SET company_id = NULL
 WHERE company_id IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM public.form_user_shares sh
                    WHERE sh.form_id = f.id AND sh.company_id = f.company_id);

-- ── Ежедневники ─────────────────────────────────────────────────
-- Ежедневник тоже лежит в пространстве. Особый вид my_day — скрытый личный
-- ежедневник экрана «Сегодня»: один на человека, в разделе «Ежедневники» его
-- нет. Его записи могут быть без даты («Потом») и нести вложения — ссылки на
-- вещи других разделов, положенные «во время».
ALTER TABLE public.diaries
    ADD COLUMN company_id bigint REFERENCES public.companies(id) ON DELETE SET NULL,
    ADD COLUMN team_access text NOT NULL DEFAULT 'edit'
        CHECK (team_access IN ('view', 'check', 'edit')),
    ADD COLUMN kind text NOT NULL DEFAULT 'regular'
        CHECK (kind IN ('regular', 'my_day')),
    ADD CONSTRAINT diaries_my_day_personal CHECK (kind = 'regular' OR company_id IS NULL);
CREATE UNIQUE INDEX diaries_my_day_uidx ON public.diaries (owner_id) WHERE kind = 'my_day';
CREATE INDEX diaries_company_idx ON public.diaries (company_id) WHERE company_id IS NOT NULL;

ALTER TABLE public.diary_records
    ALTER COLUMN entry_date DROP NOT NULL,
    ADD COLUMN attachments jsonb NOT NULL DEFAULT '[]'::jsonb
        CHECK (jsonb_typeof(attachments) = 'array');

-- ── Расписания ──────────────────────────────────────────────────
-- Участникам команды по умолчанию — просмотр: расписание обычно ведёт один.
ALTER TABLE public.schedules
    ADD COLUMN company_id bigint REFERENCES public.companies(id) ON DELETE SET NULL,
    ADD COLUMN team_access text NOT NULL DEFAULT 'view'
        CHECK (team_access IN ('view', 'edit'));
CREATE INDEX schedules_company_idx ON public.schedules (company_id) WHERE company_id IS NOT NULL;

-- ── Личные компании ─────────────────────────────────────────────
-- Их заводил вход, чтобы человеку без компании было где работать. Теперь
-- для этого есть личное пространство: компания из одного создателя без
-- задач, постов и подключений распускается, её вещи становятся личными.
CREATE TEMP TABLE dissolved ON COMMIT DROP AS
SELECT c.id FROM public.companies c
 WHERE c.name LIKE '% — личное'
   AND c.created_by IS NOT NULL
   AND NOT EXISTS (SELECT 1 FROM public.user_companies uc
                    WHERE uc.company_id = c.id AND uc.user_id <> c.created_by)
   AND NOT EXISTS (SELECT 1 FROM public.tasks t WHERE t.company_id = c.id)
   AND NOT EXISTS (SELECT 1 FROM public.portal_posts p WHERE p.company_id = c.id)
   AND NOT EXISTS (SELECT 1 FROM public.user_yougile_accounts y WHERE y.company_id = c.id)
   AND NOT EXISTS (SELECT 1 FROM public.billing_user_addons a WHERE a.company_id = c.id);

-- Переписка и звонки живут и без компании — каскад не должен их унести.
UPDATE public.conversations SET company_id = NULL WHERE company_id IN (SELECT id FROM dissolved);
UPDATE public.calls SET company_id = NULL WHERE company_id IN (SELECT id FROM dissolved);
UPDATE public.registries SET company_id = NULL WHERE company_id IN (SELECT id FROM dissolved);
UPDATE public.calendars SET company_id = NULL WHERE company_id IN (SELECT id FROM dissolved);
UPDATE public.forms SET company_id = NULL WHERE company_id IN (SELECT id FROM dissolved);
-- Платил и так создатель — меняется только пометка «файл команды».
UPDATE public.billing_storage_files SET company_id = NULL WHERE company_id IN (SELECT id FROM dissolved);
-- Шары без внешнего ключа на компании.
DELETE FROM public.note_company_shares WHERE company_id IN (SELECT id FROM dissolved);
DELETE FROM public.folder_company_shares WHERE company_id IN (SELECT id FROM dissolved);
DELETE FROM public.board_company_shares WHERE company_id IN (SELECT id FROM dissolved);
DELETE FROM public.board_folder_company_shares WHERE company_id IN (SELECT id FROM dissolved);
DELETE FROM public.companies WHERE id IN (SELECT id FROM dissolved);

-- ── Журнал биллинга ─────────────────────────────────────────────
-- Файлы переходят к новому плательщику вместе со счётчиком места: снимаем с
-- прежнего по сервисам и прибавляем новому.
CREATE TEMP TABLE space_moved ON COMMIT DROP AS
SELECT f.storage_key, f.user_id AS old_user, m.user_id AS new_user, f.service, f.size_bytes
  FROM public.billing_storage_files f
  JOIN space_moves m ON m.storage_key = f.storage_key
 WHERE f.user_id <> m.user_id;

UPDATE public.billing_storage_files f
   SET user_id = m.user_id, company_id = m.company_id
  FROM space_moves m
 WHERE f.storage_key = m.storage_key;

UPDATE public.billing_storage_usage u SET bytes = GREATEST(u.bytes - d.bytes, 0), updated_at = now()
  FROM (SELECT old_user, service, sum(size_bytes) AS bytes FROM space_moved GROUP BY 1, 2) d
 WHERE u.user_id = d.old_user AND u.service = d.service;

INSERT INTO public.billing_storage_usage (user_id, service, bytes)
SELECT new_user, service, sum(size_bytes) FROM space_moved GROUP BY 1, 2
ON CONFLICT (user_id, service) DO UPDATE
   SET bytes = public.billing_storage_usage.bytes + EXCLUDED.bytes, updated_at = now();

-- +goose Down
-- Необратимо: распущенные компании и прежняя привязка вещей восстанавливаются
-- только из резервной копии, снятой до выката.
SELECT 1;
