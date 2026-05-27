"""calls + call_participants

Revision ID: b1c2d3e4f5a6
Revises: a7b8c9d0e1f2
Create Date: 2026-05-27 18:00:00.000000

Таблицы для истории звонков. Сам сигналинг (WebRTC offer/answer/ice) и
актуальное состояние идущего звонка — in-memory (sockets/call_state.py);
БД хранит только factual log: кто кому звонил, когда, чем закончилось.
"""
from alembic import op
import sqlalchemy as sa


revision = 'b1c2d3e4f5a6'
down_revision = 'a7b8c9d0e1f2'
branch_labels = None
depends_on = None


def upgrade():
    op.create_table(
        'calls',
        sa.Column('id', sa.Integer(), primary_key=True),
        sa.Column('initiator_id', sa.Integer(), sa.ForeignKey('users.id', ondelete='CASCADE'),
                  nullable=False),
        sa.Column('kind', sa.String(length=16), nullable=False, server_default='p2p'),
        sa.Column('status', sa.String(length=16), nullable=False, server_default='ringing'),
        sa.Column('media', sa.String(length=8), nullable=False, server_default='video'),
        sa.Column('started_at', sa.DateTime(timezone=True), nullable=False,
                  server_default=sa.text('CURRENT_TIMESTAMP')),
        sa.Column('ended_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('conversation_id', sa.Integer(),
                  sa.ForeignKey('conversations.id', ondelete='SET NULL'), nullable=True),
    )
    op.create_index('idx_call_started', 'calls', ['started_at'])
    op.create_index('idx_call_status', 'calls', ['status'])

    op.create_table(
        'call_participants',
        sa.Column('id', sa.Integer(), primary_key=True),
        sa.Column('call_id', sa.Integer(), sa.ForeignKey('calls.id', ondelete='CASCADE'),
                  nullable=False),
        sa.Column('user_id', sa.Integer(), sa.ForeignKey('users.id', ondelete='CASCADE'),
                  nullable=False),
        sa.Column('role', sa.String(length=16), nullable=False, server_default='invitee'),
        sa.Column('invited_at', sa.DateTime(timezone=True), nullable=False,
                  server_default=sa.text('CURRENT_TIMESTAMP')),
        sa.Column('joined_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('left_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('declined', sa.Boolean(), nullable=False, server_default=sa.text('false')),
        sa.UniqueConstraint('call_id', 'user_id', name='uq_callpart_pair'),
    )
    op.create_index('idx_callpart_user', 'call_participants', ['user_id'])
    op.create_index('idx_callpart_call', 'call_participants', ['call_id'])


def downgrade():
    op.drop_index('idx_callpart_call', table_name='call_participants')
    op.drop_index('idx_callpart_user', table_name='call_participants')
    op.drop_table('call_participants')
    op.drop_index('idx_call_status', table_name='calls')
    op.drop_index('idx_call_started', table_name='calls')
    op.drop_table('calls')
