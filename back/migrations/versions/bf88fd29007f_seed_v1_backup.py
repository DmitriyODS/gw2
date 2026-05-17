"""seed: импорт данных из v1 backup

Revision ID: bf88fd29007f
Revises: fd7d099fb4af
Create Date: 2026-05-17 21:00:00.000000

Что делает эта миграция:
  - Создаёт 7 типов юнитов
  - Импортирует все отделы из backup.json (поле head отброшено)
  - Создаёт аккаунты пользователей (пароль сброшен до 'grovework',
    is_default_pass=TRUE → при первом входе обязательная смена)
  - Импортирует задачи: done+is_archived → is_archived=TRUE,
    new/paused → is_archived=FALSE
  - Конвертирует time_logs → units (task_type маппится на новый UnitType)
  - Все insert-ы идемпотентны (ON CONFLICT DO NOTHING)
"""
import os
import json
from alembic import op
from sqlalchemy import text


revision = 'bf88fd29007f'
down_revision = 'fd7d099fb4af'
branch_labels = None
depends_on = None

# ---------------------------------------------------------------------------
# Маппинг: старый task_type → название нового UnitType
# ---------------------------------------------------------------------------
TASK_TYPE_TO_UNIT = {
    # Дизайн
    'pub_images':                'Дизайн',
    'banner':                    'Дизайн',
    'poster':                    'Дизайн',
    'presentation':              'Дизайн',
    'presentation_update':       'Дизайн',
    'handout':                   'Дизайн',
    'stand_design':              'Дизайн',
    'pub_design':                'Дизайн',
    'branded':                   'Дизайн',
    'small_design':              'Дизайн',
    'web_design':                'Дизайн',
    'design_verify':             'Дизайн',
    'picture':                   'Дизайн',
    'revision':                  'Дизайн',
    # Текст
    'text_writing':              'Текст',
    'exports':                   'Текст',
    'surveys':                   'Текст',
    # Фото/видео
    'photo_edit':                'Фото/видео',
    'video_edit':                'Фото/видео',
    'video_shoot':               'Фото/видео',
    'photo_shoot':               'Фото/видео',
    'photo_video':               'Фото/видео',
    # Публикация
    'placement':                 'Публикация',
    'publication':               'Публикация',
    # Внутренняя работа
    'internal':                  'Внутренняя работа',
    'dept_internal':             'Внутренняя работа',
    'water_plants':              'Внутренняя работа',
    'meeting':                   'Внутренняя работа',
    'mail_work':                 'Внутренняя работа',
    'cloud_work':                'Внутренняя работа',
    'media_plan_management':     'Внутренняя работа',
    'event_calendar_management': 'Внутренняя работа',
    'other':                     'Внутренняя работа',
    # Внешняя работа
    'external':                  'Внешняя работа',
    'dept_external':             'Внешняя работа',
    # Интервью
    'interview':                 'Интервью',
}
DEFAULT_UNIT_TYPE = 'Внутренняя работа'

OLD_ROLE_TO_LEVEL = {
    'super_admin': 4,
    'admin':       3,
    'manager':     2,
    'staff':       1,
    'tv':          None,   # технический аккаунт — не мигрируем
}

UNIT_TYPES = [
    'Дизайн',
    'Текст',
    'Фото/видео',
    'Внутренняя работа',
    'Внешняя работа',
    'Публикация',
    'Интервью',
]

FALLBACK_DEPT = 'Без отдела'
DEFAULT_PASSWORD = 'grovework'


def upgrade():
    # В Docker: WORKDIR=/app (=back/), __file__=/app/migrations/versions/...
    # Два уровня вверх → /app/backup.json
    # Локально: back/migrations/versions/... → back/backup.json
    backup_path = os.path.normpath(
        os.path.join(os.path.dirname(__file__), '..', '..', 'backup.json')
    )
    if not os.path.exists(backup_path):
        print(f'  ⚠  backup.json не найден ({backup_path}), данные не импортированы')
        return

    with open(backup_path, encoding='utf-8') as fh:
        bk = json.load(fh)

    conn = op.get_bind()

    # ---------------------------------------------------------------------- #
    # 1. Типы юнитов                                                          #
    # ---------------------------------------------------------------------- #
    for name in UNIT_TYPES:
        conn.execute(
            text("INSERT INTO unit_types (name) VALUES (:n) ON CONFLICT (name) DO NOTHING"),
            {'n': name},
        )

    unit_type_id: dict[str, int] = {
        row[1]: row[0]
        for row in conn.execute(text("SELECT id, name FROM unit_types")).fetchall()
    }

    # ---------------------------------------------------------------------- #
    # 2. Резервный отдел для задач без department_id                          #
    # ---------------------------------------------------------------------- #
    conn.execute(
        text("INSERT INTO departments (name) VALUES (:n) ON CONFLICT (name) DO NOTHING"),
        {'n': FALLBACK_DEPT},
    )
    fallback_dept_id: int = conn.execute(
        text("SELECT id FROM departments WHERE name = :n"), {'n': FALLBACK_DEPT}
    ).scalar()

    # ---------------------------------------------------------------------- #
    # 3. Отделы (сохраняем оригинальные id)                                  #
    # ---------------------------------------------------------------------- #
    for dept in bk['departments']:
        conn.execute(
            text(
                "INSERT INTO departments (id, name) VALUES (:id, :n)"
                " ON CONFLICT DO NOTHING"
            ),
            {'id': dept['id'], 'n': dept['name']},
        )

    conn.execute(text(
        "SELECT setval('departments_id_seq',"
        " GREATEST((SELECT MAX(id) FROM departments), 1))"
    ))

    # ---------------------------------------------------------------------- #
    # 4. Пользователи                                                         #
    # ---------------------------------------------------------------------- #
    role_id_by_level: dict[int, int] = {
        row[1]: row[0]
        for row in conn.execute(text("SELECT id, level FROM roles")).fetchall()
    }

    user_id_map: dict[int, int] = {}   # old_id → new_id
    fallback_author_id: int | None = None

    for u in bk['users']:
        level = OLD_ROLE_TO_LEVEL.get(u['role'])
        if level is None:
            continue   # tv и другие служебные роли не мигрируем

        role_id = role_id_by_level[level]

        row = conn.execute(text("""
            INSERT INTO users
                (fio, login, hash_password, role_id, is_default_pass, is_hidden, created_at)
            VALUES
                (:fio, :login,
                 crypt(:pwd, gen_salt('bf')),
                 :role_id, TRUE, FALSE, :created_at)
            ON CONFLICT (login) DO UPDATE
                SET fio        = EXCLUDED.fio,
                    role_id    = EXCLUDED.role_id,
                    created_at = EXCLUDED.created_at
            RETURNING id
        """), {
            'fio':        u['full_name'],
            'login':      u['username'],
            'pwd':        DEFAULT_PASSWORD,
            'role_id':    role_id,
            'created_at': u['created_at'],
        }).fetchone()

        user_id_map[u['id']] = row[0]
        if fallback_author_id is None:
            fallback_author_id = row[0]

    conn.execute(text(
        "SELECT setval('users_id_seq',"
        " GREATEST((SELECT MAX(id) FROM users), 1))"
    ))

    if fallback_author_id is None:
        fallback_author_id = conn.execute(text("SELECT MIN(id) FROM users")).scalar()

    # ---------------------------------------------------------------------- #
    # 5. Задачи (сохраняем оригинальные id)                                  #
    # ---------------------------------------------------------------------- #
    task_unit_type_id: dict[int, int] = {}   # task_id → unit_type_id
    task_name_map:     dict[int, str] = {}   # task_id → title

    for t in bk['tasks']:
        dept_id   = t['department_id'] or fallback_dept_id
        author_id = user_id_map.get(t['created_by_id'], fallback_author_id)

        conn.execute(text("""
            INSERT INTO tasks
                (id, name, created_at, author_id, department_id,
                 deadline, is_archived, archived_at, received_at)
            VALUES
                (:id, :name, :created_at, :author_id, :dept_id,
                 :deadline, :is_archived, :archived_at, :received_at)
            ON CONFLICT (id) DO NOTHING
        """), {
            'id':          t['id'],
            'name':        t['title'][:500],
            'created_at':  t['created_at'],
            'author_id':   author_id,
            'dept_id':     dept_id,
            'deadline':    t.get('deadline'),
            'is_archived': bool(t['is_archived']),
            'archived_at': t.get('archived_at'),
            'received_at': t['created_at'],
        })

        ut_name = TASK_TYPE_TO_UNIT.get(t.get('task_type') or '', DEFAULT_UNIT_TYPE)
        task_unit_type_id[t['id']] = unit_type_id[ut_name]
        task_name_map[t['id']]     = t['title'][:500]

    conn.execute(text(
        "SELECT setval('tasks_id_seq',"
        " GREATEST((SELECT MAX(id) FROM tasks), 1))"
    ))

    # ---------------------------------------------------------------------- #
    # 6. Юниты (из time_logs, сохраняем оригинальные id)                     #
    # ---------------------------------------------------------------------- #
    for tl in bk['time_logs']:
        new_uid = user_id_map.get(tl['user_id'])
        if new_uid is None:
            continue   # пользователь не мигрирован

        tid = tl['task_id']
        if tid not in task_unit_type_id:
            continue   # задача не мигрирована

        conn.execute(text("""
            INSERT INTO units
                (id, name, user_id, unit_type_id, task_id,
                 is_edited, datetime_start, datetime_end, created_at)
            VALUES
                (:id, :name, :user_id, :unit_type_id, :task_id,
                 FALSE, :dt_start, :dt_end, :dt_start)
            ON CONFLICT (id) DO NOTHING
        """), {
            'id':           tl['id'],
            'name':         task_name_map[tid],
            'user_id':      new_uid,
            'unit_type_id': task_unit_type_id[tid],
            'task_id':      tid,
            'dt_start':     tl['started_at'],
            'dt_end':       tl['ended_at'],
        })

    conn.execute(text(
        "SELECT setval('units_id_seq',"
        " GREATEST((SELECT MAX(id) FROM units), 1))"
    ))

    print('  ✓  Импорт завершён: типы юнитов, отделы, пользователи, задачи, юниты')


def downgrade():
    conn = op.get_bind()
    conn.execute(text("DELETE FROM units"))
    conn.execute(text("DELETE FROM favorites"))
    conn.execute(text("DELETE FROM tasks"))
    conn.execute(text("DELETE FROM users WHERE login != 'admin'"))
    conn.execute(text("TRUNCATE departments RESTART IDENTITY CASCADE"))
    conn.execute(text("TRUNCATE unit_types RESTART IDENTITY CASCADE"))
