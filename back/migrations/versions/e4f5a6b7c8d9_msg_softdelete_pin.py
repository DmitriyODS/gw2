"""messenger soft-delete + pin

Revision ID: e4f5a6b7c8d9
Revises: d3e4f5a6b7c8
Create Date: 2026-05-25 20:00:00.000000

Удаление «только у себя» — флаги hidden_for_a/hidden_for_b на conversations и
messages. Когда обе стороны скрыли — запись физически удаляется (сервис).
Удаление «для всех» — физическое DELETE сразу.
Закрепление чатов — pinned_at_a/pinned_at_b: timestamp последнего pin со стороны.
NULL = не закреплён. Сортировка списка чатов: pinned (по времени desc) → обычные.
"""
from alembic import op
import sqlalchemy as sa


revision = 'e4f5a6b7c8d9'
down_revision = 'd3e4f5a6b7c8'
branch_labels = None
depends_on = None


def upgrade():
    with op.batch_alter_table('conversations') as b:
        b.add_column(sa.Column('hidden_for_a', sa.Boolean(), nullable=False, server_default='false'))
        b.add_column(sa.Column('hidden_for_b', sa.Boolean(), nullable=False, server_default='false'))
        b.add_column(sa.Column('pinned_at_a', sa.DateTime(timezone=True), nullable=True))
        b.add_column(sa.Column('pinned_at_b', sa.DateTime(timezone=True), nullable=True))

    op.create_index('idx_conv_pinned_a', 'conversations', ['pinned_at_a'])
    op.create_index('idx_conv_pinned_b', 'conversations', ['pinned_at_b'])

    with op.batch_alter_table('messages') as b:
        b.add_column(sa.Column('hidden_for_a', sa.Boolean(), nullable=False, server_default='false'))
        b.add_column(sa.Column('hidden_for_b', sa.Boolean(), nullable=False, server_default='false'))


def downgrade():
    with op.batch_alter_table('messages') as b:
        b.drop_column('hidden_for_b')
        b.drop_column('hidden_for_a')

    op.drop_index('idx_conv_pinned_b', table_name='conversations')
    op.drop_index('idx_conv_pinned_a', table_name='conversations')

    with op.batch_alter_table('conversations') as b:
        b.drop_column('pinned_at_b')
        b.drop_column('pinned_at_a')
        b.drop_column('hidden_for_b')
        b.drop_column('hidden_for_a')
