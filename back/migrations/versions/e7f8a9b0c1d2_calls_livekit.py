"""calls → LiveKit (комната и код ссылки-приглашения)

Revision ID: e7f8a9b0c1d2
Revises: c0d1e2f3a4b5
Create Date: 2026-06-11 09:00:00.000000

Переезд звонков на медиа-сервер LiveKit: у звонка появляется имя комнаты
LiveKit (`room_name`) и случайный код ссылки-приглашения (`share_code`),
по которому к звонку подключаются внешние гости без аккаунта.
"""
from alembic import op
import sqlalchemy as sa


revision = 'e7f8a9b0c1d2'
down_revision = 'c0d1e2f3a4b5'
branch_labels = None
depends_on = None


def upgrade():
    with op.batch_alter_table('calls') as b:
        b.add_column(sa.Column('room_name', sa.String(64), nullable=True))
        b.add_column(sa.Column('share_code', sa.String(48), nullable=True))
        b.create_unique_constraint('uq_calls_share_code', ['share_code'])


def downgrade():
    with op.batch_alter_table('calls') as b:
        b.drop_constraint('uq_calls_share_code', type_='unique')
        b.drop_column('share_code')
        b.drop_column('room_name')
