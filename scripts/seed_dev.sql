-- ================================================================
-- Демо-данные для dev-БД: компания с сотрудниками, портал с ветками
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
-- каскадом FK там, где он есть, остальное чистим явно.
DO $$
DECLARE
    demo_users bigint[];
    demo_company bigint;
BEGIN
    SELECT array_agg(id) INTO demo_users FROM users WHERE login LIKE 'demo.%';
    SELECT id INTO demo_company FROM companies WHERE name = 'Грув Демо';

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
        '{"weekend_days": [5, 6]}'::jsonb, now() - interval '90 days', FALSE);

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
    ('Объявления', 'demo.admin', 'Как мы ведём отпуска',
E'Коротко о порядке:\n\n- 📅 заявку ставим в общий календарь отдела за две недели\n- 🔁 дела передаём сменщику до ухода\n- 💬 в мессенджере включаем статус «в отпуске»\n\n> Срочное — через руководителя отдела.\n\nХорошего отдыха!',
     TRUE, 72),
    ('Релизы 🚀', 'demo.manager', 'Релиз 6.4 уехал на прод',
E'В этот раз:\n\n1. новый экран «Сегодня»\n2. ветки ответов и лайки в комментариях\n3. эмодзи в разделах портала\n\n```\nmake deploy\n```\nЖдём фидбек в комментариях.',
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

-- Обсуждение под постом про отпуска — второй тред.
WITH post AS (
    SELECT id FROM portal_posts
    WHERE company_id = (SELECT id FROM companies WHERE name = 'Грув Демо')
      AND title = 'Как мы ведём отпуска'
), root AS (
    INSERT INTO portal_comments (post_id, author_id, text, created_at)
    SELECT p.id, (SELECT id FROM users WHERE login = 'demo.pavel'),
           'А если отпуск выпадает на конец квартала? 😅', now() - interval '10 hours'
    FROM post p
    RETURNING id, post_id
)
INSERT INTO portal_comments (post_id, author_id, text, reply_to_id, created_at)
SELECT r.post_id, (SELECT id FROM users WHERE login = 'demo.elena'),
       'Тогда заранее согласуем с руководителем', r.id, now() - interval '9 hours'
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
SELECT u.login, u.fio, r.name AS role
FROM users u
JOIN user_companies uc ON uc.user_id = u.id
JOIN roles r ON r.id = uc.role_id
WHERE u.login LIKE 'demo.%'
ORDER BY r.level DESC, u.login;
