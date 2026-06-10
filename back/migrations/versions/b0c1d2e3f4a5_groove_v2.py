"""Groove v2: болезнь и характер Грувика, чат с питомцем в мессенджере

Revision ID: b0c1d2e3f4a5
Revises: a9b0c1d2e3f4
Create Date: 2026-06-10
"""
from alembic import op
import sqlalchemy as sa

revision = 'b0c1d2e3f4a5'
down_revision = 'a9b0c1d2e3f4'
branch_labels = None
depends_on = None

_CK_NEW = (
    "((is_dev_chat OR is_pet_chat) AND NOT (is_dev_chat AND is_pet_chat) "
    "    AND user_a_id IS NOT NULL AND user_b_id IS NULL) "
    "OR (NOT is_dev_chat AND NOT is_pet_chat "
    "    AND user_a_id IS NOT NULL AND user_b_id IS NOT NULL "
    "    AND user_a_id < user_b_id)"
)

_CK_OLD = (
    "(is_dev_chat AND user_a_id IS NOT NULL AND user_b_id IS NULL) "
    "OR (NOT is_dev_chat AND user_a_id IS NOT NULL AND user_b_id IS NOT NULL "
    "    AND user_a_id < user_b_id)"
)


def upgrade():
    op.add_column('pets', sa.Column('sick_since', sa.DateTime(timezone=True), nullable=True))
    op.add_column('pets', sa.Column('recovery', sa.Integer(), nullable=False, server_default='0'))
    op.add_column('pets', sa.Column('personality', sa.String(length=24), nullable=True))

    op.add_column('conversations',
                  sa.Column('is_pet_chat', sa.Boolean(), nullable=False,
                            server_default=sa.text('false')))
    op.drop_constraint('ck_conversation_pair_order', 'conversations', type_='check')
    op.create_check_constraint('ck_conversation_pair_order', 'conversations', _CK_NEW)

    op.alter_column('messages', 'sender_id', existing_type=sa.Integer(), nullable=True)
    op.add_column('messages',
                  sa.Column('is_bot', sa.Boolean(), nullable=False,
                            server_default=sa.text('false')))


def downgrade():
    op.drop_column('messages', 'is_bot')
    op.execute("DELETE FROM messages WHERE sender_id IS NULL")
    op.alter_column('messages', 'sender_id', existing_type=sa.Integer(), nullable=False)

    op.execute("DELETE FROM conversations WHERE is_pet_chat")
    op.drop_constraint('ck_conversation_pair_order', 'conversations', type_='check')
    op.create_check_constraint('ck_conversation_pair_order', 'conversations', _CK_OLD)
    op.drop_column('conversations', 'is_pet_chat')

    op.drop_column('pets', 'personality')
    op.drop_column('pets', 'recovery')
    op.drop_column('pets', 'sick_since')
