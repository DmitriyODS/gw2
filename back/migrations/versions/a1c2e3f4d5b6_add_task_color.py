"""add task.color (цвет-тег задачи)

Revision ID: a1c2e3f4d5b6
Revises: bf88fd29007f
Create Date: 2026-05-24 12:00:00.000000

Добавляет необязательное поле color — идентификатор цвета-тега из
фиксированного набора (red/orange/amber/green/teal/blue/violet/pink).
NULL означает «без цвета».
"""
from alembic import op
import sqlalchemy as sa


revision = 'a1c2e3f4d5b6'
down_revision = 'bf88fd29007f'
branch_labels = None
depends_on = None


def upgrade():
    op.add_column('tasks', sa.Column('color', sa.String(length=20), nullable=True))


def downgrade():
    op.drop_column('tasks', 'color')
