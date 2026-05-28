"""messages call link (system call entries)

Revision ID: c1d2e3f4a5b6
Revises: b1c2d3e4f5a6
Create Date: 2026-05-28 07:00:00.000000

Добавляем `messages.kind` (text|call) и `messages.call_id` (FK → calls).
Когда заканчивается p2p-звонок (как нормально, так и пропущенный), сервис
звонков создаёт в парном диалоге системное сообщение `kind='call'` с
ссылкой на запись звонка — фронт рендерит его специальной плашкой
(иконка трубки/камеры, длительность, «пропущенный»).
"""
from alembic import op
import sqlalchemy as sa


revision = 'c1d2e3f4a5b6'
down_revision = 'b1c2d3e4f5a6'
branch_labels = None
depends_on = None


def upgrade():
    with op.batch_alter_table('messages') as b:
        b.add_column(sa.Column('kind', sa.String(16), nullable=False, server_default='text'))
        b.add_column(sa.Column('call_id', sa.Integer(), nullable=True))
        b.create_foreign_key('fk_msg_call', 'calls', ['call_id'], ['id'],
                             ondelete='SET NULL')


def downgrade():
    with op.batch_alter_table('messages') as b:
        b.drop_constraint('fk_msg_call', type_='foreignkey')
        b.drop_column('call_id')
        b.drop_column('kind')
