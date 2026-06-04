"""v3 — companies, multi-tenancy, новые роли, контакты

Revision ID: e1c2d3a4b5f6
Revises: d1e2f3a4b5c6
Create Date: 2026-06-04 12:00:00.000000

Корневая миграция перехода на v3.0:
  * таблица `companies` (name, description, director_id, is_active, settings, created_at);
  * добавление `company_id` в users/tasks/units/departments/unit_types/conversations/calls;
  * добавление `is_root_admin`, `phone`, `email` в users;
  * переименование ролей в БД: «Администратор»→«Руководитель»,
    «Суперадминистратор»→«Администратор» (уровни 1..4 не меняются);
  * создание дефолтной компании «Главная компания» и привязка к ней всех
    существующих сущностей;
  * проставление is_root_admin=TRUE для пользователя с минимальным id среди
    обладателей level=4 (бывший корневой суперадминистратор).

Уникальность departments/unit_types меняется с глобальной (UNIQUE name) на
уникальную в рамках компании (UNIQUE (company_id, name)) — у разных компаний
могут быть одноимённые отделы.
"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql


revision = 'e1c2d3a4b5f6'
down_revision = 'd1e2f3a4b5c6'
branch_labels = None
depends_on = None


DEFAULT_COMPANY_NAME = 'Главная компания'
DEFAULT_SETTINGS_JSON = (
    '{"uses_yougile": true, "uses_stages": false, "uses_calls": true}'
)


def upgrade():
    # 1. companies
    op.create_table(
        'companies',
        sa.Column('id', sa.Integer(), primary_key=True),
        sa.Column('name', sa.String(length=255), nullable=False),
        sa.Column('description', sa.Text(), nullable=True),
        # director_id — FK на users, но users появится ниже; добавим
        # после, через add_column + create_foreign_key.
        sa.Column('director_id', sa.Integer(), nullable=True),
        sa.Column('is_active', sa.Boolean(), nullable=False, server_default=sa.text('TRUE')),
        sa.Column('settings', postgresql.JSONB(astext_type=sa.Text()),
                  nullable=False, server_default=sa.text("'{}'::jsonb")),
        sa.Column('created_at', sa.DateTime(timezone=True), nullable=False,
                  server_default=sa.text('NOW()')),
        sa.UniqueConstraint('name', name='uq_companies_name'),
    )
    op.create_index('idx_companies_active', 'companies', ['is_active'])

    # 2. companies.director_id → users.id (FK добавляем после users-обновлений)
    # ничего сейчас, FK добавим в конце.

    # 3. users: company_id + is_root_admin + phone + email
    with op.batch_alter_table('users') as b:
        b.add_column(sa.Column('company_id', sa.Integer(), nullable=True))
        b.add_column(sa.Column('phone', sa.String(length=20), nullable=True))
        b.add_column(sa.Column('email', sa.String(length=255), nullable=True))
        b.add_column(sa.Column('is_root_admin', sa.Boolean(),
                               nullable=False, server_default=sa.text('FALSE')))
    op.create_foreign_key('fk_users_company', 'users', 'companies',
                          ['company_id'], ['id'], ondelete='SET NULL')
    op.create_index('idx_users_company', 'users', ['company_id'])
    # Уникальный email (case-insensitive), NULL разрешён несколькими записями
    op.execute("""
        CREATE UNIQUE INDEX uq_users_email_lower
        ON users (LOWER(email))
        WHERE email IS NOT NULL
    """)

    # 4. departments / unit_types / tasks / units / conversations / calls — company_id
    for tbl in ('departments', 'unit_types', 'tasks', 'units', 'conversations', 'calls'):
        with op.batch_alter_table(tbl) as b:
            b.add_column(sa.Column('company_id', sa.Integer(), nullable=True))

    # 5. Дефолтная компания + миграция данных
    #    Создаём дефолтную компанию; директор пока NULL — назначим ниже.
    op.execute(f"""
        INSERT INTO companies (name, description, is_active, settings, created_at)
        VALUES ('{DEFAULT_COMPANY_NAME}', 'Создана автоматически при переходе на v3.0',
                TRUE, '{DEFAULT_SETTINGS_JSON}'::jsonb, NOW())
    """)

    # Берём id дефолтной компании прямой подстановкой через SELECT
    op.execute("""
        UPDATE users SET company_id = (SELECT id FROM companies WHERE name = '"""
               + DEFAULT_COMPANY_NAME + """') WHERE company_id IS NULL""")
    op.execute("""
        UPDATE departments SET company_id = (SELECT id FROM companies WHERE name = '"""
               + DEFAULT_COMPANY_NAME + """') WHERE company_id IS NULL""")
    op.execute("""
        UPDATE unit_types SET company_id = (SELECT id FROM companies WHERE name = '"""
               + DEFAULT_COMPANY_NAME + """') WHERE company_id IS NULL""")
    op.execute("""
        UPDATE tasks SET company_id = (SELECT id FROM companies WHERE name = '"""
               + DEFAULT_COMPANY_NAME + """') WHERE company_id IS NULL""")
    op.execute("""
        UPDATE units SET company_id = (SELECT id FROM companies WHERE name = '"""
               + DEFAULT_COMPANY_NAME + """') WHERE company_id IS NULL""")
    op.execute("""
        UPDATE conversations SET company_id = (SELECT id FROM companies WHERE name = '"""
               + DEFAULT_COMPANY_NAME + """') WHERE company_id IS NULL""")
    op.execute("""
        UPDATE calls SET company_id = (SELECT id FROM companies WHERE name = '"""
               + DEFAULT_COMPANY_NAME + """') WHERE company_id IS NULL""")

    # 6. Сначала ставим NOT NULL и FK там, где company_id должен быть обязательным.
    #    На users company_id остаётся NULLABLE (Администратор системы — без компании).
    op.execute("ALTER TABLE departments ALTER COLUMN company_id SET NOT NULL")
    op.execute("ALTER TABLE unit_types ALTER COLUMN company_id SET NOT NULL")
    op.execute("ALTER TABLE tasks ALTER COLUMN company_id SET NOT NULL")
    op.execute("ALTER TABLE units ALTER COLUMN company_id SET NOT NULL")
    op.execute("ALTER TABLE conversations ALTER COLUMN company_id SET NOT NULL")
    op.execute("ALTER TABLE calls ALTER COLUMN company_id SET NOT NULL")

    op.create_foreign_key('fk_departments_company', 'departments', 'companies',
                          ['company_id'], ['id'], ondelete='CASCADE')
    op.create_foreign_key('fk_unit_types_company', 'unit_types', 'companies',
                          ['company_id'], ['id'], ondelete='CASCADE')
    op.create_foreign_key('fk_tasks_company', 'tasks', 'companies',
                          ['company_id'], ['id'], ondelete='CASCADE')
    op.create_foreign_key('fk_units_company', 'units', 'companies',
                          ['company_id'], ['id'], ondelete='CASCADE')
    op.create_foreign_key('fk_conversations_company', 'conversations', 'companies',
                          ['company_id'], ['id'], ondelete='CASCADE')
    op.create_foreign_key('fk_calls_company', 'calls', 'companies',
                          ['company_id'], ['id'], ondelete='CASCADE')

    op.create_index('idx_departments_company', 'departments', ['company_id'])
    op.create_index('idx_unit_types_company', 'unit_types', ['company_id'])
    op.create_index('idx_tasks_company', 'tasks', ['company_id'])
    op.create_index('idx_units_company', 'units', ['company_id'])
    op.create_index('idx_conv_company', 'conversations', ['company_id'])
    op.create_index('idx_call_company', 'calls', ['company_id'])

    # 7. Уникальность departments/unit_types: глобальный UNIQUE → per-company
    op.execute("ALTER TABLE departments DROP CONSTRAINT IF EXISTS departments_name_key")
    op.execute("ALTER TABLE unit_types DROP CONSTRAINT IF EXISTS unit_types_name_key")
    op.create_unique_constraint('uq_departments_company_name', 'departments',
                                ['company_id', 'name'])
    op.create_unique_constraint('uq_unit_types_company_name', 'unit_types',
                                ['company_id', 'name'])

    # 8. Переименование ролей: «Суперадминистратор» → «Администратор»,
    #    «Администратор» → «Руководитель». Делаем в одной транзакции,
    #    через промежуточные имена, чтобы UNIQUE(name) не сломался.
    op.execute("UPDATE roles SET name = '__tmp_admin__' WHERE name = 'Администратор'")
    op.execute("UPDATE roles SET name = 'Администратор' WHERE name = 'Суперадминистратор'")
    op.execute("UPDATE roles SET name = 'Руководитель' WHERE name = '__tmp_admin__'")

    # 9. Корневой Администратор системы — первый по id обладатель level=4.
    op.execute("""
        UPDATE users SET is_root_admin = TRUE
        WHERE id = (
            SELECT u.id FROM users u JOIN roles r ON u.role_id = r.id
            WHERE r.level = 4 ORDER BY u.id ASC LIMIT 1
        )
    """)

    # 10. Дефолтная компания: директором ставим первого по id обладателя level=3
    #     (бывший «Администратор», теперь «Руководитель»). Если такого нет —
    #     директор остаётся NULL (можно назначить вручную из UI).
    op.execute(f"""
        UPDATE companies SET director_id = (
            SELECT u.id FROM users u JOIN roles r ON u.role_id = r.id
            WHERE r.level = 3 AND u.is_hidden = FALSE
            ORDER BY u.id ASC LIMIT 1
        ) WHERE name = '{DEFAULT_COMPANY_NAME}'
    """)

    # 11. Корневые Администраторы системы не должны быть привязаны к компании.
    op.execute("UPDATE users SET company_id = NULL WHERE is_root_admin = TRUE")

    # 12. Теперь добавляем FK companies.director_id → users.id
    op.create_foreign_key('fk_companies_director', 'companies', 'users',
                          ['director_id'], ['id'], ondelete='SET NULL')


def downgrade():
    op.drop_constraint('fk_companies_director', 'companies', type_='foreignkey')

    op.execute("UPDATE roles SET name = '__tmp_dir__' WHERE name = 'Руководитель'")
    op.execute("UPDATE roles SET name = 'Суперадминистратор' WHERE name = 'Администратор'")
    op.execute("UPDATE roles SET name = 'Администратор' WHERE name = '__tmp_dir__'")

    op.drop_constraint('uq_unit_types_company_name', 'unit_types', type_='unique')
    op.drop_constraint('uq_departments_company_name', 'departments', type_='unique')
    op.create_unique_constraint('unit_types_name_key', 'unit_types', ['name'])
    op.create_unique_constraint('departments_name_key', 'departments', ['name'])

    company_indexes = {
        'calls': 'idx_call_company',
        'conversations': 'idx_conv_company',
        'units': 'idx_units_company',
        'tasks': 'idx_tasks_company',
        'unit_types': 'idx_unit_types_company',
        'departments': 'idx_departments_company',
    }
    for tbl in ('calls', 'conversations', 'units', 'tasks', 'unit_types', 'departments'):
        op.drop_index(company_indexes[tbl], table_name=tbl)
        op.drop_constraint(f'fk_{tbl}_company', tbl, type_='foreignkey')
        with op.batch_alter_table(tbl) as b:
            b.drop_column('company_id')

    op.execute("DROP INDEX IF EXISTS uq_users_email_lower")
    op.drop_index('idx_users_company', table_name='users')
    op.drop_constraint('fk_users_company', 'users', type_='foreignkey')
    with op.batch_alter_table('users') as b:
        b.drop_column('is_root_admin')
        b.drop_column('email')
        b.drop_column('phone')
        b.drop_column('company_id')

    op.drop_index('idx_companies_active', table_name='companies')
    op.drop_table('companies')
