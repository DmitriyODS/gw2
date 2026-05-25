"""user_task_colors: индивидуальные цвета карточек на пользователя

Revision ID: c2d3e4f5a6b7
Revises: a1c2e3f4d5b6
Create Date: 2026-05-25 09:00:00.000000

Создаёт таблицу user_task_colors: один и тот же task у разных пользователей
может быть окрашен в разный цвет (или вовсе не окрашен — записи нет).
Поле tasks.color больше не используется фронтом, но удалять колонку не
будем — миграция данных переносит её значения в новую таблицу для всех
пользователей, кто потом сможет «получить наследство» как личный цвет.
В нашем случае переносить как глобальный нельзя, поэтому колонка просто
остаётся как технический архив до следующей чистки.
"""
from alembic import op
import sqlalchemy as sa


revision = 'c2d3e4f5a6b7'
down_revision = 'a1c2e3f4d5b6'
branch_labels = None
depends_on = None


def upgrade():
    op.create_table(
        'user_task_colors',
        sa.Column('user_id', sa.Integer(), nullable=False),
        sa.Column('task_id', sa.Integer(), nullable=False),
        sa.Column('color', sa.String(length=20), nullable=False),
        sa.ForeignKeyConstraint(['user_id'], ['users.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['task_id'], ['tasks.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('user_id', 'task_id'),
    )
    op.create_index('idx_utc_task', 'user_task_colors', ['task_id'])


def downgrade():
    op.drop_index('idx_utc_task', table_name='user_task_colors')
    op.drop_table('user_task_colors')
