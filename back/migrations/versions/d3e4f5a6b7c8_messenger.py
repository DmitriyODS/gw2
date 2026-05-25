"""messenger: conversations, messages, message_attachments

Revision ID: d3e4f5a6b7c8
Revises: c2d3e4f5a6b7
Create Date: 2026-05-25 18:00:00.000000

Личный мессенджер 1:1. Диалог хранится как пара (user_a < user_b), уникальная
независимо от инициатора. Сообщение может содержать текст и/или вложения.
Вложение создаётся отдельным upload-запросом (получает id), затем привязывается
к сообщению при отправке.
"""
from alembic import op
import sqlalchemy as sa


revision = 'd3e4f5a6b7c8'
down_revision = 'c2d3e4f5a6b7'
branch_labels = None
depends_on = None


def upgrade():
    op.create_table(
        'conversations',
        sa.Column('id', sa.Integer(), primary_key=True),
        sa.Column('user_a_id', sa.Integer(), nullable=False),
        sa.Column('user_b_id', sa.Integer(), nullable=False),
        sa.Column('created_at', sa.DateTime(timezone=True), nullable=False,
                  server_default=sa.text('CURRENT_TIMESTAMP')),
        sa.Column('last_message_at', sa.DateTime(timezone=True), nullable=True),
        sa.ForeignKeyConstraint(['user_a_id'], ['users.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['user_b_id'], ['users.id'], ondelete='CASCADE'),
        sa.UniqueConstraint('user_a_id', 'user_b_id', name='uq_conversation_pair'),
        sa.CheckConstraint('user_a_id < user_b_id', name='ck_conversation_pair_order'),
    )
    op.create_index('idx_conv_user_a', 'conversations', ['user_a_id'])
    op.create_index('idx_conv_user_b', 'conversations', ['user_b_id'])
    op.create_index('idx_conv_last_msg', 'conversations', ['last_message_at'])

    op.create_table(
        'messages',
        sa.Column('id', sa.Integer(), primary_key=True),
        sa.Column('conversation_id', sa.Integer(), nullable=False),
        sa.Column('sender_id', sa.Integer(), nullable=False),
        sa.Column('text', sa.Text(), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), nullable=False,
                  server_default=sa.text('CURRENT_TIMESTAMP')),
        sa.Column('read_at', sa.DateTime(timezone=True), nullable=True),
        sa.ForeignKeyConstraint(['conversation_id'], ['conversations.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['sender_id'], ['users.id'], ondelete='CASCADE'),
    )
    op.create_index('idx_msg_conv_created', 'messages', ['conversation_id', 'created_at'])
    op.create_index('idx_msg_unread_recipient', 'messages', ['conversation_id', 'read_at'])

    op.create_table(
        'message_attachments',
        sa.Column('id', sa.Integer(), primary_key=True),
        sa.Column('message_id', sa.Integer(), nullable=True),
        sa.Column('uploader_id', sa.Integer(), nullable=False),
        sa.Column('file_path', sa.String(length=500), nullable=False),
        sa.Column('file_name', sa.String(length=255), nullable=False),
        sa.Column('mime_type', sa.String(length=120), nullable=False),
        sa.Column('size_bytes', sa.Integer(), nullable=False),
        sa.Column('created_at', sa.DateTime(timezone=True), nullable=False,
                  server_default=sa.text('CURRENT_TIMESTAMP')),
        sa.ForeignKeyConstraint(['message_id'], ['messages.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['uploader_id'], ['users.id'], ondelete='CASCADE'),
    )
    op.create_index('idx_att_message', 'message_attachments', ['message_id'])


def downgrade():
    op.drop_index('idx_att_message', table_name='message_attachments')
    op.drop_table('message_attachments')
    op.drop_index('idx_msg_unread_recipient', table_name='messages')
    op.drop_index('idx_msg_conv_created', table_name='messages')
    op.drop_table('messages')
    op.drop_index('idx_conv_last_msg', table_name='conversations')
    op.drop_index('idx_conv_user_b', table_name='conversations')
    op.drop_index('idx_conv_user_a', table_name='conversations')
    op.drop_table('conversations')
