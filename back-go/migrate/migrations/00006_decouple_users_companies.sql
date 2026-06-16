-- +goose Up
-- Развязка идентичности и компаний. users — чистый аккаунт: он не знает про
-- компании. Принадлежность и роль живут ТОЛЬКО в user_companies (связке).
-- Бывший «Администратор системы» (company_id IS NULL, is_root_admin) становится
-- платформенным супер-админом (users.is_super_admin) — отдельным классом, без
-- доступа к компанийной функциональности. Компания самодостаточна: её настройки
-- уже в companies, а «руководитель» — это членство с ролью «Администратор».

-- 1. users: новые поля идентичности.
ALTER TABLE public.users ADD COLUMN is_super_admin boolean NOT NULL DEFAULT false;
ALTER TABLE public.users ADD COLUMN is_active boolean NOT NULL DEFAULT true;

-- Корневой Администратор системы → единственный супер-админ (логин/пароль сохраняются).
UPDATE public.users SET is_super_admin = true WHERE is_root_admin = true;
-- Глобально отключённый аккаунт = прежний is_hidden.
UPDATE public.users SET is_active = NOT is_hidden;

-- 2. user_companies — единственный источник принадлежности: должность переезжает сюда.
ALTER TABLE public.user_companies ADD COLUMN post character varying(255);
UPDATE public.user_companies uc
   SET post = u.post
  FROM public.users u
 WHERE u.id = uc.user_id AND u.company_id = uc.company_id AND u.post IS NOT NULL;

-- 3. companies: создатель для аудита (без особых прав — администраторы равны).
ALTER TABLE public.companies ADD COLUMN created_by integer REFERENCES public.users(id) ON DELETE SET NULL;
UPDATE public.companies SET created_by = director_id WHERE director_id IS NOT NULL;

-- Гарантия: бывший руководитель компании имеет членство с ролью «Администратор»
-- (роль уровня 3; её id == 3 по фиксированному сиду ролей).
INSERT INTO public.user_companies (user_id, company_id, role_id, created_at)
SELECT c.director_id, c.id, 3, now()
  FROM public.companies c
 WHERE c.director_id IS NOT NULL
ON CONFLICT (user_id, company_id) DO UPDATE SET role_id = 3;

ALTER TABLE public.companies DROP COLUMN director_id;

-- 4. Переписка и звонки между людьми без общей компании: company_id опционален.
ALTER TABLE public.conversations ALTER COLUMN company_id DROP NOT NULL;
ALTER TABLE public.calls ALTER COLUMN company_id DROP NOT NULL;

-- 5. users теряет всё компанийное (индексы и FK на этих колонках уходят каскадом).
-- Делаем это ДО чистки ролей: иначе users.role_id (FK) держит системную роль.
ALTER TABLE public.users
    DROP COLUMN company_id,
    DROP COLUMN role_id,
    DROP COLUMN post,
    DROP COLUMN is_hidden,
    DROP COLUMN is_root_admin;

-- 6. roles — три роли компании. Системную роль (level 4) удаляем (её носители
-- стали супер-админом, членств не имеют — FK из user_companies не держит), затем
-- сид трёх ролей. Удаление ПЕРВЫМ освобождает имя «Администратор» для уровня 3.
DELETE FROM public.roles r WHERE r.level >= 4
  AND NOT EXISTS (SELECT 1 FROM public.user_companies uc WHERE uc.role_id = r.id);
INSERT INTO public.roles (id, name, level) VALUES
    (1, 'Сотрудник', 1),
    (2, 'Менеджер', 2),
    (3, 'Администратор', 3)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, level = EXCLUDED.level;

-- +goose Down
-- Внимание: откат — best-effort. Часть данных восстанавливается приблизительно
-- (primary-компания — первое членство), а company_id переписок/звонков может
-- содержать NULL и тогда SET NOT NULL не пройдёт — это ожидаемо для dev-отката.
ALTER TABLE public.users ADD COLUMN company_id integer;
ALTER TABLE public.users ADD COLUMN role_id integer;
ALTER TABLE public.users ADD COLUMN post character varying(255);
ALTER TABLE public.users ADD COLUMN is_hidden boolean NOT NULL DEFAULT false;
ALTER TABLE public.users ADD COLUMN is_root_admin boolean NOT NULL DEFAULT false;

UPDATE public.users u
   SET company_id = uc.company_id, role_id = uc.role_id, post = uc.post
  FROM (SELECT DISTINCT ON (user_id) user_id, company_id, role_id, post
          FROM public.user_companies ORDER BY user_id, created_at) uc
 WHERE uc.user_id = u.id;
UPDATE public.users SET is_root_admin = is_super_admin;
UPDATE public.users SET is_hidden = NOT is_active;

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_company_id_fkey FOREIGN KEY (company_id)
    REFERENCES public.companies(id) ON DELETE SET NULL;
CREATE INDEX idx_users_company ON public.users USING btree (company_id);
CREATE INDEX idx_users_role ON public.users USING btree (role_id);

ALTER TABLE public.companies ADD COLUMN director_id integer
    REFERENCES public.users(id) ON DELETE SET NULL;
UPDATE public.companies SET director_id = created_by;
ALTER TABLE public.companies DROP COLUMN created_by;

ALTER TABLE public.conversations ALTER COLUMN company_id SET NOT NULL;
ALTER TABLE public.calls ALTER COLUMN company_id SET NOT NULL;

ALTER TABLE public.user_companies DROP COLUMN post;
ALTER TABLE public.users DROP COLUMN is_super_admin;
ALTER TABLE public.users DROP COLUMN is_active;
