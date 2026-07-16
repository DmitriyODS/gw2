-- ================================================================
-- Демо-данные для dev-БД: компания с сотрудниками, грувики во ВСЕХ
-- состояниях (здоровый, голодный, простуженный, грязнуля, хандра на
-- пороге побега, одинокий, в приключении), портал с ветками
-- комментариев и лайками, задачи с юнитами для статистики.
--
-- Идемпотентен: повторный запуск чистит прежний посев (компания
-- «Грув Демо» и её пользователи с логинами demo.*) и создаёт заново.
-- Пароль всех демо-аккаунтов: demo1234
--
-- Запуск: make dev-seed  (или scripts/seed_dev.sh)
-- ================================================================
BEGIN;

-- ── Чистка прошлого посева ──────────────────────────────────────
-- Пользователи demo.* и компания «Грув Демо»: связанные строки уходят
-- каскадом FK там, где он есть, остальное чистим явно (те же таблицы,
-- что и в DeletePet — они ссылаются на users, а не на pets).
DO $$
DECLARE
    demo_users bigint[];
    demo_company bigint;
BEGIN
    SELECT array_agg(id) INTO demo_users FROM users WHERE login LIKE 'demo.%';
    SELECT id INTO demo_company FROM companies WHERE name = 'Грув Демо';

    IF demo_users IS NOT NULL THEN
        DELETE FROM pet_strokes WHERE pet_user_id = ANY(demo_users) OR user_id = ANY(demo_users);
        DELETE FROM pet_shop_purchases WHERE user_id = ANY(demo_users);
        DELETE FROM pet_kudos_weekly WHERE user_id = ANY(demo_users);
        DELETE FROM pet_kudos_seasonal WHERE user_id = ANY(demo_users);
        DELETE FROM pet_season_claims WHERE user_id = ANY(demo_users);
        DELETE FROM pet_kudos_ledger WHERE user_id = ANY(demo_users);
        DELETE FROM pets WHERE user_id = ANY(demo_users);
    END IF;
    IF demo_company IS NOT NULL THEN
        DELETE FROM units WHERE task_id IN (SELECT id FROM tasks WHERE company_id = demo_company);
        DELETE FROM tasks WHERE company_id = demo_company;
        DELETE FROM unit_types WHERE company_id = demo_company;
        DELETE FROM departments WHERE company_id = demo_company;
        DELETE FROM portal_posts WHERE company_id = demo_company;
        DELETE FROM portal_topics WHERE company_id = demo_company;
        DELETE FROM companies WHERE id = demo_company;
    END IF;
    IF demo_users IS NOT NULL THEN
        DELETE FROM users WHERE id = ANY(demo_users);
    END IF;
END $$;

-- ── Компания ────────────────────────────────────────────────────
INSERT INTO companies (name, description, is_active, settings, created_at, ai_enabled)
VALUES ('Грув Демо', 'Демо-компания с данными для проверки', TRUE,
        '{"weekend_days": [5, 6], "uses_groove": true}'::jsonb, now() - interval '90 days', FALSE);

-- ── Сотрудники ──────────────────────────────────────────────────
-- created_by компании проставим после вставки админа (см. ниже).
INSERT INTO users (fio, login, hash_password, is_default_pass, created_at, email,
                   is_super_admin, is_active, email_verified, status_emoji, status_text)
VALUES
    ('Демидова Анна Петровна',   'demo.admin',   crypt('demo1234', gen_salt('bf')), FALSE, now() - interval '90 days', 'demo.admin@example.com',   FALSE, TRUE, TRUE, '🎯', 'Рулю компанией'),
    ('Морозов Игорь Сергеевич',  'demo.manager', crypt('demo1234', gen_salt('bf')), FALSE, now() - interval '85 days', 'demo.manager@example.com', FALSE, TRUE, TRUE, '📊', 'Считаю метрики'),
    ('Соколова Мария Ивановна',  'demo.maria',   crypt('demo1234', gen_salt('bf')), FALSE, now() - interval '80 days', 'demo.maria@example.com',   FALSE, TRUE, TRUE, '🚀', 'В потоке'),
    ('Кузнецов Павел Юрьевич',   'demo.pavel',   crypt('demo1234', gen_salt('bf')), FALSE, now() - interval '75 days', 'demo.pavel@example.com',   FALSE, TRUE, TRUE, NULL, NULL),
    ('Волкова Ольга Дмитриевна', 'demo.olga',    crypt('demo1234', gen_salt('bf')), FALSE, now() - interval '70 days', 'demo.olga@example.com',    FALSE, TRUE, TRUE, '☕', 'Кофе-брейк'),
    ('Лебедев Артём Русланович', 'demo.artem',   crypt('demo1234', gen_salt('bf')), FALSE, now() - interval '65 days', 'demo.artem@example.com',   FALSE, TRUE, TRUE, NULL, NULL),
    ('Новиков Денис Олегович',   'demo.denis',   crypt('demo1234', gen_salt('bf')), FALSE, now() - interval '60 days', 'demo.denis@example.com',   FALSE, TRUE, TRUE, NULL, NULL),
    ('Зайцева Елена Андреевна',  'demo.elena',   crypt('demo1234', gen_salt('bf')), FALSE, now() - interval '55 days', 'demo.elena@example.com',   FALSE, TRUE, TRUE, '🌙', 'Сова');

UPDATE companies SET created_by = (SELECT id FROM users WHERE login = 'demo.admin')
WHERE name = 'Грув Демо';

-- Членство: администратор-создатель, менеджер, остальные — сотрудники.
INSERT INTO user_companies (user_id, company_id, role_id, created_at, post)
SELECT u.id, c.id,
       CASE u.login WHEN 'demo.admin' THEN 3 WHEN 'demo.manager' THEN 2 ELSE 1 END,
       now() - interval '60 days',
       CASE u.login
           WHEN 'demo.admin' THEN 'Руководитель'
           WHEN 'demo.manager' THEN 'Менеджер проектов'
           WHEN 'demo.maria' THEN 'Аналитик'
           WHEN 'demo.pavel' THEN 'Разработчик'
           WHEN 'demo.olga' THEN 'Дизайнер'
           WHEN 'demo.artem' THEN 'Разработчик'
           WHEN 'demo.denis' THEN 'Тестировщик'
           ELSE 'Поддержка'
       END
FROM users u CROSS JOIN companies c
WHERE u.login LIKE 'demo.%' AND c.name = 'Грув Демо';

-- ── Задачи и юниты (статистика, «в эфире», лечение хандры работой) ──
INSERT INTO unit_types (name, company_id)
SELECT t.name, c.id FROM companies c,
     (VALUES ('Разработка'), ('Созвон'), ('Ревью')) AS t(name)
WHERE c.name = 'Грув Демо';

INSERT INTO departments (name, company_id)
SELECT d.name, c.id FROM companies c,
     (VALUES ('Продукт'), ('Разработка'), ('Поддержка')) AS d(name)
WHERE c.name = 'Грув Демо';

INSERT INTO tasks (name, created_at, author_id, received_at, is_archived, company_id,
                   department_id, responsible_user_id)
SELECT t.name, now() - (t.age || ' days')::interval,
       (SELECT id FROM users WHERE login = 'demo.admin'),
       now() - (t.age || ' days')::interval, t.archived,
       (SELECT id FROM companies WHERE name = 'Грув Демо'),
       (SELECT id FROM departments
        WHERE company_id = (SELECT id FROM companies WHERE name = 'Грув Демо')
          AND name = t.dept),
       (SELECT id FROM users WHERE login = t.who)
FROM (VALUES
    ('Свёрстать личный кабинет',        12, FALSE, 'demo.olga',    'Продукт'),
    ('Починить экспорт отчёта',          9, TRUE,  'demo.pavel',   'Разработка'),
    ('Разобрать бэклог поддержки',       7, FALSE, 'demo.elena',   'Поддержка'),
    ('Прогнать регресс перед релизом',   5, TRUE,  'demo.denis',   'Разработка'),
    ('Собрать аналитику по воронке',     4, FALSE, 'demo.maria',   'Продукт'),
    ('Обновить документацию API',        2, FALSE, 'demo.artem',   'Разработка'),
    ('Подготовить демо для заказчика',   1, FALSE, 'demo.manager', 'Продукт')
) AS t(name, age, archived, who, dept);

-- Завершённые юниты за последние две недели: дают статистику, характер
-- питомца и «последнюю работу» (от неё считается хандра).
INSERT INTO units (name, user_id, company_id, unit_type_id, task_id, is_edited,
                   datetime_start, datetime_end, created_at)
SELECT 'Работа над задачей',
       t.responsible_user_id, t.company_id,
       (SELECT id FROM unit_types WHERE company_id = t.company_id ORDER BY id LIMIT 1),
       t.id, FALSE,
       now() - (d.day || ' days')::interval - interval '3 hours',
       now() - (d.day || ' days')::interval - interval '3 hours' + ((30 + (t.id * 7 + d.day * 13) % 90) || ' minutes')::interval,
       now() - (d.day || ' days')::interval
FROM tasks t
CROSS JOIN generate_series(1, 6) AS d(day)
WHERE t.company_id = (SELECT id FROM companies WHERE name = 'Грув Демо')
  AND t.responsible_user_id IS NOT NULL
  -- У «хандрящего» Дениса работы давно не было — иначе болезнь не сойдётся.
  AND t.responsible_user_id <> (SELECT id FROM users WHERE login = 'demo.denis');

-- Активный юнит Марии — блок «Сейчас в эфире».
INSERT INTO units (name, user_id, company_id, unit_type_id, task_id, is_edited,
                   datetime_start, datetime_end, created_at)
SELECT 'Считаю воронку',
       (SELECT id FROM users WHERE login = 'demo.maria'), t.company_id,
       (SELECT id FROM unit_types WHERE company_id = t.company_id ORDER BY id LIMIT 1),
       t.id, FALSE, now() - interval '25 minutes', NULL, now() - interval '25 minutes'
FROM tasks t
WHERE t.company_id = (SELECT id FROM companies WHERE name = 'Грув Демо')
  AND t.name = 'Собрать аналитику по воронке';

-- ── Грувики во всех состояниях ──────────────────────────────────
-- Каждый демо-питомец показывает свою механику: здоровый, голодный,
-- простуженный, грязнуля, хандра на пороге побега, одинокий и путник.
INSERT INTO pets (user_id, company_id, name, species, stage, xp, kudos, hat, accessories,
                  feed_streak, last_fed_date, sick_since, ailment, recovery,
                  need_satiety, need_energy, need_hygiene, need_social, needs_at,
                  personality, unlocked_species, quest_progress, quest_claimed,
                  adventure_until, adventure_place, generation, house_owned, house_placed,
                  house_theme, created_at)
SELECT
    u.id, c.id, p.pet_name, p.species, p.stage, p.xp, p.kudos, p.hat, p.accessories::jsonb,
    p.streak,
    CASE WHEN p.fed_days_ago IS NULL THEN NULL
         ELSE (now() - (p.fed_days_ago || ' days')::interval)::date END,
    CASE WHEN p.sick_days IS NULL THEN NULL
         ELSE now() - (p.sick_days || ' days')::interval END,
    p.ailment, p.recovery,
    p.satiety, p.energy, p.hygiene, p.social, now(),
    p.personality, p.unlocked::jsonb, 0, FALSE,
    NULL, NULL, -- приключение ставится отдельным UPDATE (см. ниже)
    p.generation, p.house_owned::jsonb, p.house_placed::jsonb, p.theme,
    now() - interval '60 days'
FROM companies c
JOIN (VALUES
    -- Здоровый прокачанный: эталон «всё хорошо», настроение отличное.
    ('demo.admin',   'Босс',     'owl',        5, 1020, 640, 'crown',  '["crown","tie","medal"]', 9,  0,    NULL, NULL,     0, 95,  90, 88, 80, 'steady',    '["owl","fox"]',        2, '["sofa","piano","plant","picture"]', '[{"key":"sofa","x":30,"y":70},{"key":"piano","x":68,"y":66},{"key":"plant","x":14,"y":74},{"key":"picture","x":50,"y":24}]', 'night'),
    ('demo.manager', 'Метрик',   'marathoner', 4, 620,  310, 'glasses','["glasses","tie"]',       5,  0,    NULL, NULL,     0, 78,  64, 70, 55, 'zen',       '["marathoner"]',       1, '["chair","books"]', '[{"key":"chair","x":34,"y":72},{"key":"books","x":66,"y":70}]', 'cozy'),
    -- Голодный: сытость в нуле → истощение. Лечит еда (бульон).
    ('demo.pavel',   'Крош',     'sprinter',   3, 300,  45,  NULL,     '["cap"]',                 0,  3,    2,    'hunger', 0, 0,   52, 60, 40, 'energizer', '["sprinter"]',         1, '[]', '[]', 'cozy'),
    -- Простуженный: выдохся, энергия на нуле. Лечит сон.
    ('demo.olga',    'Пиксель',  'lark',       3, 340,  120, 'bow',    '["bow","flower"]',        2,  1,    1,    'cold',   1, 55,  0,  62, 66, 'early',     '["lark"]',             1, '["bed","teddy"]', '[{"key":"bed","x":28,"y":70},{"key":"teddy","x":62,"y":72}]', 'lavender'),
    -- Грязнуля: чистота в нуле. Лечит купание — одного раза хватит.
    ('demo.artem',   'Уголёк',   'fox',        2, 160,  80,  NULL,     '["headphones"]',          1,  1,    2,    'grime',  0, 60,  58, 0,  45, 'steady',    '["fox"]',              1, '[]', '[]', 'forest'),
    -- Хандра 13-й день: завтра сбежит — проверка предупреждения и побега.
    ('demo.denis',   'Ждун',     'marathoner', 4, 700,  260, 'helmet', '["helmet"]',              0,  9,    13,   'blues',  1, 44,  50, 48, 20, 'lazy',      '["marathoner","fox"]', 1, '["console"]', '[{"key":"console","x":50,"y":72}]', 'space'),
    -- Одинокий и здоровый: общения почти нет → повод погладить.
    ('demo.elena',   'Тихоня',   'owl',        3, 380,  95,  'scarf',  '["scarf","mittens"]',     3,  0,    NULL, NULL,     0, 72,  68, 74, 5,  'night',     '["owl"]',              1, '["plant"]', '[{"key":"plant","x":52,"y":72}]', 'ocean'),
    -- В приключении: платные действия отвечают PET_AWAY, на виджете 🧭.
    ('demo.maria',   'Комета',   'unicorn',    4, 580,  400, 'star',   '["star","rainbow"]',      6,  0,    NULL, NULL,     0, 82,  74, 80, 70, 'energizer', '["unicorn","sprinter"]', 1, '["fountain","garland"]', '[{"key":"fountain","x":40,"y":68},{"key":"garland","x":50,"y":18}]', 'candy')
) AS p(login, pet_name, species, stage, xp, kudos, hat, accessories, streak, fed_days_ago,
       sick_days, ailment, recovery, satiety, energy, hygiene, social, personality, unlocked,
       generation, house_owned, house_placed, theme)
  ON TRUE
JOIN users u ON u.login = p.login
WHERE c.name = 'Грув Демо';

-- Комета — в приключении (поля берём отдельным UPDATE: в VALUES выше их
-- нет, чтобы не плодить NULL-колонки у остальных).
UPDATE pets SET adventure_until = now() + interval '2 hours', adventure_place = 'в горы'
WHERE user_id = (SELECT id FROM users WHERE login = 'demo.maria');

-- Недельный рейтинг признания: у каждого своя строка — топ не пустой.
INSERT INTO pet_kudos_weekly (user_id, iso_year, iso_week, amount)
SELECT u.id, EXTRACT(isoyear FROM now())::int, EXTRACT(week FROM now())::int, w.amount
FROM (VALUES
    ('demo.admin', 64), ('demo.maria', 51), ('demo.manager', 38),
    ('demo.olga', 27), ('demo.artem', 19), ('demo.pavel', 12),
    ('demo.elena', 8), ('demo.denis', 3)
) AS w(login, amount)
JOIN users u ON u.login = w.login;

-- Сезонный трек (квартал МСК): у админа открыты первые пороги.
INSERT INTO pet_kudos_seasonal (user_id, season, amount)
SELECT u.id,
       EXTRACT(year FROM now())::text || '-Q' || EXTRACT(quarter FROM now())::text,
       s.amount
FROM (VALUES ('demo.admin', 420), ('demo.maria', 260), ('demo.manager', 130)) AS s(login, amount)
JOIN users u ON u.login = s.login;

-- Выписка банка: приход/расход за две недели — история и аналитика.
INSERT INTO pet_kudos_ledger (user_id, company_id, delta, kind, counterparty_id, comment, created_at)
SELECT u.id, c.id, l.delta, l.kind,
       CASE WHEN l.cp IS NULL THEN NULL ELSE (SELECT id FROM users WHERE login = l.cp) END,
       l.comment, now() - (l.days || ' days')::interval
FROM companies c
JOIN (VALUES
    ('demo.admin',   12,  'unit',        NULL,          '',              1),
    ('demo.admin',   25,  'task_closed', NULL,          '',              2),
    ('demo.admin',   -10, 'feed',        NULL,          '',              2),
    ('demo.admin',   20,  'quest',       NULL,          '',              3),
    ('demo.admin',   -30, 'transfer_out','demo.elena',  'за помощь',     3),
    ('demo.admin',   3,   'stroke_in',   'demo.maria',  '',              4),
    ('demo.admin',   -60, 'house',       NULL,          '',              5),
    ('demo.maria',   15,  'unit',        NULL,          '',              1),
    ('demo.maria',   -2,  'stroke',      'demo.admin',  '',              4),
    ('demo.maria',   8,   'adventure',   NULL,          'на речку',      2),
    ('demo.maria',   -12, 'bath',        NULL,          '',              3),
    ('demo.elena',   30,  'transfer_in', 'demo.admin',  'за помощь',     3),
    ('demo.elena',   -25, 'heal',        NULL,          '',              6),
    ('demo.pavel',   -1,  'feed',        NULL,          'бульон',        1),
    ('demo.olga',    -15, 'walk',        NULL,          '',              2)
) AS l(login, delta, kind, cp, comment, days) ON TRUE
JOIN users u ON u.login = l.login
WHERE c.name = 'Грув Демо';

-- Поглаживания за сегодня: у Кометы уже 2 из 3 от админа — виден лимит.
INSERT INTO pet_strokes (pet_user_id, user_id, day, created_at)
SELECT (SELECT id FROM users WHERE login = 'demo.maria'),
       (SELECT id FROM users WHERE login = 'demo.admin'),
       (now() AT TIME ZONE 'utc')::date, now() - (n || ' hours')::interval
FROM generate_series(1, 2) AS n;

-- ── Портал: разделы (иконки И эмодзи), посты, ветки, лайки ──────
INSERT INTO portal_topics (company_id, name, color, icon, created_by, created_at)
SELECT c.id, t.name, t.color, t.icon,
       (SELECT id FROM users WHERE login = 'demo.admin'), now() - interval '50 days'
FROM companies c
JOIN (VALUES
    ('Объявления',        'blue',   'campaign'),
    ('Релизы 🚀',         'violet', 'rocket_launch'),
    ('Кухня и кофе',      'amber',  '☕'),               -- эмодзи вместо иконки
    ('Спорт 🏆',          'teal',   '🏃'),               -- эмодзи и в названии, и вместо иконки
    ('База знаний',       'indigo', 'menu_book')
) AS t(name, color, icon) ON TRUE
WHERE c.name = 'Грув Демо';

INSERT INTO portal_posts (company_id, topic_id, author_id, title, body,
                          pinned_at, pinned_by, created_at, updated_at)
SELECT c.id,
       (SELECT id FROM portal_topics WHERE company_id = c.id AND name = p.topic),
       (SELECT id FROM users WHERE login = p.author),
       p.title, p.body,
       CASE WHEN p.pinned THEN now() - interval '3 days' ELSE NULL END,
       CASE WHEN p.pinned THEN (SELECT id FROM users WHERE login = 'demo.admin') ELSE NULL END,
       now() - (p.hours || ' hours')::interval,
       now() - (p.hours || ' hours')::interval
FROM companies c
JOIN (VALUES
    ('Объявления', 'demo.admin', 'Правила заботы о грувиках',
E'Коротко о новом: у грувиков появились **потребности**.\n\n- 🍖 сытость — тает за сутки, голодный слегает с истощением\n- ⚡ энергия — восполняется сном\n- 🫧 чистота — купание в тазике\n- 💬 общение — закрывают коллеги, когда гладят питомца\n\n> Заброшенный грувик через две недели болезни сбежит, и прогресс обнулится.\n\nГладьте друг друга — хозяину капают кудосы.',
     TRUE, 72),
    ('Релизы 🚀', 'demo.manager', 'Релиз 6.4 уехал на прод',
E'В этот раз:\n\n1. потребности и болезни грувиков\n2. ветки ответов и лайки в комментариях\n3. эмодзи в разделах портала\n\n```\nmake deploy\n```\nЖдём фидбек в комментариях.',
     FALSE, 30),
    ('Кухня и кофе', 'demo.olga', 'Кофемашину починили ☕',
E'Работает как новая. Капучино снова с пенкой, эспрессо — без драмы.\n\n| Напиток | Кнопка |\n|---|---|\n| Эспрессо | 1 |\n| Капучино | 2 |',
     FALSE, 20),
    ('Спорт 🏆', 'demo.elena', 'Забег в субботу — кто с нами?',
E'Стартуем в 10:00 у парка. Дистанции 5 и 10 км.\n\n- [x] маршрут согласован\n- [ ] футболки\n- [ ] фотограф',
     FALSE, 12),
    ('База знаний', 'demo.maria', 'Как читать воронку в аналитике',
E'Три шага:\n\n1. открыть раздел «Статистика»\n2. выбрать период\n3. смотреть на конверсию между этапами\n\nПодробности — в комментариях, задавайте вопросы.',
     FALSE, 6),
    (NULL, 'demo.artem', NULL,
E'Кто-нибудь видел мою кружку? Синяя, с надписью «Deploy on Friday».',
     FALSE, 2)
) AS p(topic, author, title, body, pinned, hours) ON TRUE
WHERE c.name = 'Грув Демо';

-- Комментарии деревом: корень → ответ → ответ на ответ (три уровня).
WITH post AS (
    SELECT id FROM portal_posts
    WHERE company_id = (SELECT id FROM companies WHERE name = 'Грув Демо')
      AND title = 'Релиз 6.4 уехал на прод'
), root AS (
    INSERT INTO portal_comments (post_id, author_id, text, created_at)
    SELECT p.id, (SELECT id FROM users WHERE login = 'demo.olga'),
           'Наконец-то ветки в комментариях! Спасибо 🎉', now() - interval '20 hours'
    FROM post p
    RETURNING id, post_id
), reply1 AS (
    INSERT INTO portal_comments (post_id, author_id, text, reply_to_id, created_at)
    SELECT r.post_id, (SELECT id FROM users WHERE login = 'demo.manager'),
           'И лайки — теперь видно, что коллеги согласны, без десяти «+1»', r.id,
           now() - interval '19 hours'
    FROM root r
    RETURNING id, post_id
), reply2 AS (
    INSERT INTO portal_comments (post_id, author_id, text, reply_to_id, created_at)
    SELECT r.post_id, (SELECT id FROM users WHERE login = 'demo.pavel'),
           'Подтверждаю, читать стало сильно проще', r.id, now() - interval '18 hours'
    FROM reply1 r
    RETURNING id, post_id
), reply3 AS (
    INSERT INTO portal_comments (post_id, author_id, text, reply_to_id, created_at)
    SELECT r.post_id, (SELECT id FROM users WHERE login = 'demo.admin'),
           'Глубина не ограничена — но не увлекайтесь 🙂', r.id, now() - interval '17 hours'
    FROM reply2 r
    RETURNING id, post_id
), other AS (
    INSERT INTO portal_comments (post_id, author_id, text, created_at)
    SELECT p.id, (SELECT id FROM users WHERE login = 'demo.denis'),
           'А регресс перед релизом прогнали?', now() - interval '16 hours'
    FROM post p
    RETURNING id, post_id
)
INSERT INTO portal_comments (post_id, author_id, text, reply_to_id, created_at)
SELECT o.post_id, (SELECT id FROM users WHERE login = 'demo.manager'),
       'Прогнали, всё зелёное', o.id, now() - interval '15 hours'
FROM other o;

-- Обсуждение под постом про грувиков — второй тред.
WITH post AS (
    SELECT id FROM portal_posts
    WHERE company_id = (SELECT id FROM companies WHERE name = 'Грув Демо')
      AND title = 'Правила заботы о грувиках'
), root AS (
    INSERT INTO portal_comments (post_id, author_id, text, created_at)
    SELECT p.id, (SELECT id FROM users WHERE login = 'demo.pavel'),
           'Мой Крош уже слёг от голода 😅 чем кормить-то?', now() - interval '10 hours'
    FROM post p
    RETURNING id, post_id
)
INSERT INTO portal_comments (post_id, author_id, text, reply_to_id, created_at)
SELECT r.post_id, (SELECT id FROM users WHERE login = 'demo.elena'),
       'Больного кормят бульоном — он дешёвый, кнопка та же', r.id, now() - interval '9 hours'
FROM root r;

-- Лайки: у корневых комментариев — по несколько, чтобы счётчик был живым.
INSERT INTO portal_comment_likes (comment_id, user_id, created_at)
SELECT c.id, u.id, now() - interval '8 hours'
FROM portal_comments c
JOIN users u ON u.login IN ('demo.admin', 'demo.maria', 'demo.artem', 'demo.elena')
WHERE c.reply_to_id IS NULL
  AND c.post_id IN (SELECT id FROM portal_posts
                    WHERE company_id = (SELECT id FROM companies WHERE name = 'Грув Демо'))
ON CONFLICT DO NOTHING;

INSERT INTO portal_comment_likes (comment_id, user_id, created_at)
SELECT c.id, (SELECT id FROM users WHERE login = 'demo.admin'), now() - interval '7 hours'
FROM portal_comments c
WHERE c.reply_to_id IS NOT NULL
  AND c.post_id IN (SELECT id FROM portal_posts
                    WHERE company_id = (SELECT id FROM companies WHERE name = 'Грув Демо'))
ON CONFLICT DO NOTHING;

-- Просмотры постов: счётчик на карточках не пустой.
INSERT INTO portal_post_views (post_id, user_id, viewed_at)
SELECT p.id, u.id, now() - interval '5 hours'
FROM portal_posts p
JOIN users u ON u.login LIKE 'demo.%'
WHERE p.company_id = (SELECT id FROM companies WHERE name = 'Грув Демо')
ON CONFLICT DO NOTHING;

COMMIT;

-- ── Что получилось ──────────────────────────────────────────────
SELECT u.login, u.fio, r.name AS role, p.name AS pet, p.stage,
       p.ailment, p.need_satiety AS "сытость", p.need_energy AS "энергия",
       p.need_hygiene AS "чистота", p.need_social AS "общение", p.kudos
FROM users u
JOIN user_companies uc ON uc.user_id = u.id
JOIN roles r ON r.id = uc.role_id
LEFT JOIN pets p ON p.user_id = u.id
WHERE u.login LIKE 'demo.%'
ORDER BY r.level DESC, u.login;
