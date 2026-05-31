"""message pin (закрепление сообщений в диалоге)

Revision ID: d1e2f3a4b5c6
Revises: c1d2e3f4a5b6
Create Date: 2026-05-31 09:00:00.000000

Добавляем `messages.pinned_at` (момент закрепления) и `messages.pinned_by_id`
(кто закрепил, SET NULL при удалении пользователя). Закрепление общее для
обоих участников диалога — закреплённое сообщение видят оба (как в Telegram).
"""
from alembic import op
import sqlalchemy as sa


revision = 'd1e2f3a4b5c6'
down_revision = 'c1d2e3f4a5b6'
branch_labels = None
depends_on = None


def upgrade():
    with op.batch_alter_table('messages') as b:
        b.add_column(sa.Column('pinned_at', sa.DateTime(timezone=True), nullable=True))
        b.add_column(sa.Column('pinned_by_id', sa.Integer(), nullable=True))
        b.create_foreign_key('fk_msg_pinned_by', 'users', ['pinned_by_id'], ['id'],
                             ondelete='SET NULL')
    op.create_index('idx_msg_pinned', 'messages', ['conversation_id', 'pinned_at'])


def downgrade():
    op.drop_index('idx_msg_pinned', table_name='messages')
    with op.batch_alter_table('messages') as b:
        b.drop_constraint('fk_msg_pinned_by', type_='foreignkey')
        b.drop_column('pinned_by_id')
        b.drop_column('pinned_at')
