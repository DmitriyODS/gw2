"""Фундамент интеграции с YouGile

Revision ID: c3d4e5f6a7b8
Revises: b3c4d5e6f7a8
Create Date: 2026-06-07 10:00:00.000000

Этап 1 интеграции с YouGile (см. CLAUDE.md разделы по плану): добавляем
структуры, которые нужны самой синхронизации, но пока не дёргаем ни один
YouGile-эндпоинт. Этот шаг безопасно катится в прод — фич-флага не нужно,
старый код продолжает работать с `tasks.link_yougile` как со свободной
ссылкой.

Изменения:

1) `companies` — параметры подключения на уровне компании. Идея в том, что
   admin фиксирует пару (YG-компания, проект, доска) один раз, а пользователи
   только цепляют свой логин/пароль. Поэтому всё хранится тут, не у юзера:

   - yg_company_id / yg_company_name — выбранная компания в YouGile.
   - yg_project_id / yg_project_title — выбранный проект.
   - yg_board_id / yg_board_title — выбранная доска.
   - yg_first_column_id — куда летят новые задачи (резолвится из /columns).
   - yg_completed_column_id — опц., если задана, GW-задачи при архивации
     переезжают в эту колонку YG.
   - yg_webhook_id / yg_webhook_secret — наш ингресс webhook (secret в URL
     для простого shared-secret сценария).

2) `user_yougile_accounts` — личный коннект пользователя к YouGile. 1:1 к
   user_id. `key_ciphertext` шифруется Fernet'ом (YOUGILE_ENC_KEY), хранить
   plain нельзя. `key_fingerprint` (last4) показывается в UI.

3) `tasks` — связь со конкретной карточкой YouGile:
   - yougile_task_id — id карточки в YG (UUID). Уникален в пределах компании.
   - yougile_project_id/board_id/column_id — для бейджа и маппинга статусов.
   - yougile_synced_at / yougile_sync_hash — антицикл: при исходящем PUT
     запоминаем хеш применённого state'а, и если webhook вернул тот же —
     игнорируем (иначе вечный пинг-понг).

Поле `link_yougile` оставляем как было — это URL для UI, его всегда можно
кликнуть; новые структурированные поля живут рядом и заполняются автоматом
при привязке через API.
"""
from alembic import op
import sqlalchemy as sa


revision = 'c3d4e5f6a7b8'
down_revision = 'b3c4d5e6f7a8'
branch_labels = None
depends_on = None


def upgrade():
    # ── 1. companies ──────────────────────────────────────────────────────
    with op.batch_alter_table('companies') as batch:
        batch.add_column(sa.Column('yg_company_id', sa.String(length=64), nullable=True))
        batch.add_column(sa.Column('yg_company_name', sa.String(length=255), nullable=True))
        batch.add_column(sa.Column('yg_project_id', sa.String(length=64), nullable=True))
        batch.add_column(sa.Column('yg_project_title', sa.String(length=255), nullable=True))
        batch.add_column(sa.Column('yg_board_id', sa.String(length=64), nullable=True))
        batch.add_column(sa.Column('yg_board_title', sa.String(length=255), nullable=True))
        batch.add_column(sa.Column('yg_first_column_id', sa.String(length=64), nullable=True))
        batch.add_column(sa.Column('yg_completed_column_id', sa.String(length=64), nullable=True))
        batch.add_column(sa.Column('yg_webhook_id', sa.String(length=64), nullable=True))
        batch.add_column(sa.Column('yg_webhook_secret', sa.String(length=64), nullable=True))

    # ── 2. user_yougile_accounts ──────────────────────────────────────────
    op.create_table(
        'user_yougile_accounts',
        sa.Column('id', sa.Integer(), primary_key=True),
        sa.Column('user_id', sa.Integer(), nullable=False),
        sa.Column('company_id', sa.Integer(), nullable=False),
        sa.Column('yg_company_id', sa.String(length=64), nullable=False),
        sa.Column('yg_user_id', sa.String(length=64), nullable=True),
        sa.Column('yg_login', sa.String(length=255), nullable=False),
        sa.Column('key_ciphertext', sa.LargeBinary(), nullable=False),
        sa.Column('key_fingerprint', sa.String(length=8), nullable=False),
        sa.Column('last_validated_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True),
                  nullable=False, server_default=sa.text('now()')),
        sa.Column('updated_at', sa.DateTime(timezone=True),
                  nullable=False, server_default=sa.text('now()')),
        sa.ForeignKeyConstraint(['user_id'], ['users.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['company_id'], ['companies.id'], ondelete='CASCADE'),
        sa.UniqueConstraint('user_id', name='uq_user_yg_account'),
    )
    op.create_index('idx_user_yg_company', 'user_yougile_accounts', ['company_id'])

    # ── 3. tasks ─────────────────────────────────────────────────────────
    with op.batch_alter_table('tasks') as batch:
        batch.add_column(sa.Column('yougile_task_id', sa.String(length=64), nullable=True))
        batch.add_column(sa.Column('yougile_project_id', sa.String(length=64), nullable=True))
        batch.add_column(sa.Column('yougile_board_id', sa.String(length=64), nullable=True))
        batch.add_column(sa.Column('yougile_column_id', sa.String(length=64), nullable=True))
        batch.add_column(sa.Column('yougile_synced_at', sa.DateTime(timezone=True), nullable=True))
        batch.add_column(sa.Column('yougile_sync_hash', sa.String(length=64), nullable=True))

    # Уникальность yougile_task_id внутри компании: одна и та же YG-карточка
    # не должна быть привязана к двум GW-задачам. Частичный индекс — пустой
    # task_id (NULL) обходит ограничение.
    op.create_index(
        'uq_tasks_yougile_per_company',
        'tasks',
        ['company_id', 'yougile_task_id'],
        unique=True,
        postgresql_where=sa.text('yougile_task_id IS NOT NULL'),
    )


def downgrade():
    op.drop_index('uq_tasks_yougile_per_company', table_name='tasks')
    with op.batch_alter_table('tasks') as batch:
        batch.drop_column('yougile_sync_hash')
        batch.drop_column('yougile_synced_at')
        batch.drop_column('yougile_column_id')
        batch.drop_column('yougile_board_id')
        batch.drop_column('yougile_project_id')
        batch.drop_column('yougile_task_id')

    op.drop_index('idx_user_yg_company', table_name='user_yougile_accounts')
    op.drop_table('user_yougile_accounts')

    with op.batch_alter_table('companies') as batch:
        batch.drop_column('yg_webhook_secret')
        batch.drop_column('yg_webhook_id')
        batch.drop_column('yg_completed_column_id')
        batch.drop_column('yg_first_column_id')
        batch.drop_column('yg_board_title')
        batch.drop_column('yg_board_id')
        batch.drop_column('yg_project_title')
        batch.drop_column('yg_project_id')
        batch.drop_column('yg_company_name')
        batch.drop_column('yg_company_id')
