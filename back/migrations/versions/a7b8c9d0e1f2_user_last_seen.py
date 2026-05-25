"""user last_seen_at (онлайн-статус)

Revision ID: a7b8c9d0e1f2
Revises: f5a6b7c8d9e0
Create Date: 2026-05-25 23:30:00.000000

Время последнего выхода из сети — обновляется при дисконнекте всех сокетов
пользователя. Онлайн-статус (кто сейчас в сети) держится в памяти процесса и
рассылается через WebSocket; в БД хранится только last_seen для оффлайн.
"""
from alembic import op
import sqlalchemy as sa


revision = 'a7b8c9d0e1f2'
down_revision = 'f5a6b7c8d9e0'
branch_labels = None
depends_on = None


def upgrade():
    op.add_column('users', sa.Column('last_seen_at', sa.DateTime(timezone=True), nullable=True))


def downgrade():
    op.drop_column('users', 'last_seen_at')
