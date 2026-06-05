"""v3 — мессенджер: спец-чаты компаний с разработчиками + прикрепление задач

Revision ID: b5c6d7e8f9a0
Revises: a3b4c5d6e7f8
Create Date: 2026-06-05 09:00:00.000000

Этап 4 v3.0:
- conversations.is_dev_chat BOOLEAN — флаг спец-чата компании. У dev-чата
  user_a_id и user_b_id = NULL (поэтому делаем колонки nullable). Уникальность
  спец-чата на компанию — partial unique index.
- conversations.user_a_id/user_b_id — становятся nullable, чтобы dev-чаты
  не нарушали NOT NULL. Старая check-констрейнт `user_a_id < user_b_id`
  заменяется на условную: `is_dev_chat OR (user_a_id < user_b_id)`.
- messages.task_id (FK tasks SET NULL) — прикреплённая задача.
- messages.kind — расширяем по семантике: 'text' | 'call' | 'task' |
  'system_dev_reply' (ответ разработчиков в dev-чате). Изменений типа колонки
  не требуется (VARCHAR(16)).
"""
from alembic import op
import sqlalchemy as sa


revision = 'b5c6d7e8f9a0'
down_revision = 'a3b4c5d6e7f8'
branch_labels = None
depends_on = None


def upgrade():
    # conversations: is_dev_chat + nullable user pair + переоформление check
    op.add_column(
        'conversations',
        sa.Column('is_dev_chat', sa.Boolean(), nullable=False,
                  server_default=sa.text('false')),
    )
    op.alter_column('conversations', 'user_a_id', existing_type=sa.Integer(), nullable=True)
    op.alter_column('conversations', 'user_b_id', existing_type=sa.Integer(), nullable=True)

    op.drop_constraint('ck_conversation_pair_order', 'conversations', type_='check')
    op.create_check_constraint(
        'ck_conversation_pair_order',
        'conversations',
        'is_dev_chat OR (user_a_id IS NOT NULL AND user_b_id IS NOT NULL AND user_a_id < user_b_id)',
    )

    # Старый UNIQUE на пару допускал только одну NULL-NULL запись — для dev-чатов
    # этого мало (нам нужно один dev-чат на компанию). Снимаем общий UNIQUE и
    # ставим два partial-индекса: на пары обычных диалогов и на dev-чат компании.
    op.drop_constraint('uq_conversation_pair', 'conversations', type_='unique')
    op.create_index(
        'uq_conversation_pair', 'conversations', ['user_a_id', 'user_b_id'],
        unique=True, postgresql_where=sa.text('is_dev_chat = FALSE'),
    )
    op.create_index(
        'uq_conversation_dev_company', 'conversations', ['company_id'],
        unique=True, postgresql_where=sa.text('is_dev_chat = TRUE'),
    )

    # messages.task_id
    with op.batch_alter_table('messages') as b:
        b.add_column(sa.Column('task_id', sa.Integer(), nullable=True))
        b.create_foreign_key(
            'fk_msg_task_id', 'tasks', ['task_id'], ['id'], ondelete='SET NULL',
        )
    op.create_index('idx_msg_task', 'messages', ['task_id'])


def downgrade():
    op.drop_index('idx_msg_task', table_name='messages')
    with op.batch_alter_table('messages') as b:
        b.drop_constraint('fk_msg_task_id', type_='foreignkey')
        b.drop_column('task_id')

    op.drop_index('uq_conversation_dev_company', table_name='conversations')
    op.drop_index('uq_conversation_pair', table_name='conversations')
    op.create_unique_constraint('uq_conversation_pair', 'conversations',
                                ['user_a_id', 'user_b_id'])

    op.drop_constraint('ck_conversation_pair_order', 'conversations', type_='check')
    op.create_check_constraint('ck_conversation_pair_order', 'conversations',
                               'user_a_id < user_b_id')

    op.alter_column('conversations', 'user_b_id', existing_type=sa.Integer(), nullable=False)
    op.alter_column('conversations', 'user_a_id', existing_type=sa.Integer(), nullable=False)
    op.drop_column('conversations', 'is_dev_chat')
