"""messenger reply + forward

Revision ID: f5a6b7c8d9e0
Revises: e4f5a6b7c8d9
Create Date: 2026-05-25 22:00:00.000000

Ответы на сообщения — reply_to_id (FK messages.id, SET NULL при удалении
оригинала, чтобы ответ не каскадно-удалялся, а лишь потерял цитату).
Пересылка — forwarded_from_user_id (FK users.id, SET NULL): автор оригинала
для метки «Переслано от …». При пересылке текст/файлы физически копируются.
"""
from alembic import op
import sqlalchemy as sa


revision = 'f5a6b7c8d9e0'
down_revision = 'e4f5a6b7c8d9'
branch_labels = None
depends_on = None


def upgrade():
    with op.batch_alter_table('messages') as b:
        b.add_column(sa.Column('reply_to_id', sa.Integer(), nullable=True))
        b.add_column(sa.Column('forwarded_from_user_id', sa.Integer(), nullable=True))
        b.create_foreign_key('fk_msg_reply_to', 'messages', ['reply_to_id'], ['id'],
                             ondelete='SET NULL')
        b.create_foreign_key('fk_msg_forwarded_from', 'users', ['forwarded_from_user_id'], ['id'],
                             ondelete='SET NULL')


def downgrade():
    with op.batch_alter_table('messages') as b:
        b.drop_constraint('fk_msg_forwarded_from', type_='foreignkey')
        b.drop_constraint('fk_msg_reply_to', type_='foreignkey')
        b.drop_column('forwarded_from_user_id')
        b.drop_column('reply_to_id')
