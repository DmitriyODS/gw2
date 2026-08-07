-- +goose Up
-- Личная компания каждому аккаунту.
--
-- К компании привязана вся работа (задачи, юниты, статистика, реестры,
-- календари, портал), поэтому человек без компании попадал в приложение, где
-- половина разделов отвечает «нужна активная компания». Теперь такая компания
-- есть у всех: у новых она заводится при первом входе (startSession), у
-- существующих — этой миграцией.
--
-- Супер-администратор пропускается: он платформенный, к рабочим данным
-- компаний доступа не имеет и своей компании иметь не должен.
--
-- Название — «Фамилия Имя — личное»; при совпадении добавляется номер, потому
-- что имена компаний уникальны, а тёзки неизбежны.
-- +goose StatementBegin
DO $$
DECLARE
    u          RECORD;
    base       text;
    candidate  text;
    attempt    int;
    new_id     bigint;
    admin_role int;
BEGIN
    SELECT id INTO admin_role FROM public.roles WHERE level = 3 LIMIT 1;
    IF admin_role IS NULL THEN
        RAISE EXCEPTION 'роль администратора не найдена — миграция ролей должна быть применена раньше';
    END IF;

    FOR u IN
        SELECT us.id, us.fio
          FROM public.users us
         WHERE NOT us.is_super_admin
           AND NOT EXISTS (SELECT 1 FROM public.user_companies uc WHERE uc.user_id = us.id)
    LOOP
        base := NULLIF(trim(array_to_string((string_to_array(trim(u.fio), ' '))[1:2], ' ')), '');
        base := COALESCE(base, 'Моя компания') || ' — личное';

        candidate := base;
        attempt := 2;
        WHILE EXISTS (SELECT 1 FROM public.companies c WHERE c.name = candidate) LOOP
            candidate := replace(base, ' — личное', '') || ' ' || attempt || ' — личное';
            attempt := attempt + 1;
        END LOOP;

        INSERT INTO public.companies (name, created_by, is_active)
        VALUES (candidate, u.id, TRUE)
        RETURNING id INTO new_id;

        INSERT INTO public.user_companies (user_id, company_id, role_id)
        VALUES (u.id, new_id, admin_role);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- Возврата нет: какие компании были личными, а какие человек завёл сам,
-- различить уже нельзя — удаление снесло бы и настоящие данные.
SELECT 1;
