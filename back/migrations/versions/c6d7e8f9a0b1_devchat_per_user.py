"""Чат техподдержки — per user (а не per company)

Revision ID: c6d7e8f9a0b1
Revises: b5c6d7e8f9a0
Create Date: 2026-06-04 12:00:00.000000

Меняем модель спец-чата:
- Раньше: один dev-чат на компанию (UNIQUE company_id), user_a/user_b = NULL,
  его видели все сотрудники компании + все Администраторы системы.
- Теперь: один dev-чат на пользователя — это его личный чат с техподдержкой.
  user_a_id = id пользователя, user_b_id = NULL. UNIQUE (user_a_id).
  Видят: только владелец и все Администраторы системы (root admin'ы).

Существующие dev-чаты по согласованию удаляются (вместе с сообщениями и
вложениями — каскадом). Файлы-вложения на диске остаются как мусор; их
можно подмести вручную при необходимости (миграция не лезет в файловую
систему, чтобы не зависеть от состояния UPLOAD_FOLDER).
"""
from alembic import op
import sqlalchemy as sa


revision = 'c6d7e8f9a0b1'
down_revision = 'b5c6d7e8f9a0'
branch_labels = None
depends_on = None


def upgrade():
    # 1. Удаляем все старые dev-чаты — каскад снесёт сообщения и пиннинг.
    op.execute("DELETE FROM conversations WHERE is_dev_chat = TRUE")

    # 2. Снимаем UNIQUE на (company_id) WHERE is_dev_chat — он больше не нужен.
    op.drop_index('uq_conversation_dev_company', table_name='conversations')

    # 3. CHECK constraint: dev-чат теперь должен иметь user_a_id (владельца)
    #    и user_b_id = NULL. Обычный диалог — как и раньше.
    op.drop_constraint('ck_conversation_pair_order', 'conversations', type_='check')
    op.create_check_constraint(
        'ck_conversation_pair_order',
        'conversations',
        '(is_dev_chat AND user_a_id IS NOT NULL AND user_b_id IS NULL) '
        'OR (NOT is_dev_chat AND user_a_id IS NOT NULL AND user_b_id IS NOT NULL '
        '    AND user_a_id < user_b_id)',
    )

    # 4. Уникальный индекс: у одного пользователя — ровно один dev-чат.
    op.create_index(
        'uq_conversation_dev_user', 'conversations', ['user_a_id'],
        unique=True, postgresql_where=sa.text('is_dev_chat = TRUE'),
    )


def downgrade():
    # Возвращаем старую модель «один dev-чат на компанию». Существующие
    # per-user dev-чаты сносим (их нельзя автоматически смержить в один на
    # компанию без потери привязки сообщений к авторам/контексту).
    op.execute("DELETE FROM conversations WHERE is_dev_chat = TRUE")

    op.drop_index('uq_conversation_dev_user', table_name='conversations')

    op.drop_constraint('ck_conversation_pair_order', 'conversations', type_='check')
    op.create_check_constraint(
        'ck_conversation_pair_order',
        'conversations',
        'is_dev_chat OR (user_a_id IS NOT NULL AND user_b_id IS NOT NULL '
        'AND user_a_id < user_b_id)',
    )

    op.create_index(
        'uq_conversation_dev_company', 'conversations', ['company_id'],
        unique=True, postgresql_where=sa.text('is_dev_chat = TRUE'),
    )
