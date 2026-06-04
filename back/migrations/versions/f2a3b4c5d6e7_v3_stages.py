"""v3 — stages (этапы задач)

Revision ID: f2a3b4c5d6e7
Revises: e1c2d3a4b5f6
Create Date: 2026-06-04 16:00:00.000000

Создаёт таблицу `stages` (этапы задач, привязаны к компании). Используется
канбан-режимом задач (третье представление, появляется в Этапе 3) и в новом
разделе «Списки» (Этап 2).
"""
from alembic import op
import sqlalchemy as sa


revision = 'f2a3b4c5d6e7'
down_revision = 'e1c2d3a4b5f6'
branch_labels = None
depends_on = None


def upgrade():
    op.create_table(
        'stages',
        sa.Column('id', sa.Integer(), primary_key=True),
        sa.Column('company_id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=255), nullable=False),
        sa.Column('color', sa.String(length=16), nullable=False, server_default='blue'),
        sa.Column('order', sa.Integer(), nullable=False, server_default='0'),
        sa.ForeignKeyConstraint(['company_id'], ['companies.id'], ondelete='CASCADE'),
        sa.UniqueConstraint('company_id', 'name', name='uq_stages_company_name'),
    )
    op.create_index('idx_stages_company_order', 'stages', ['company_id', 'order'])


def downgrade():
    op.drop_index('idx_stages_company_order', table_name='stages')
    op.drop_table('stages')
