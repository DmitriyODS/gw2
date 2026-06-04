"""v3 — задачи: ответственный, этап, комментарии

Revision ID: a3b4c5d6e7f8
Revises: f2a3b4c5d6e7
Create Date: 2026-06-04 17:00:00.000000

Этап 3 v3.0:
- tasks.responsible_user_id (FK users SET NULL) — у старых задач NULL,
  у новых проставляется автоматически (= автор).
- tasks.stage_id (FK stages SET NULL) — текущий этап задачи.
- Таблица `comments`: id, task_id, author_id, text (MD), created_at,
  updated_at, deleted_at (soft-delete).
"""
from alembic import op
import sqlalchemy as sa


revision = 'a3b4c5d6e7f8'
down_revision = 'f2a3b4c5d6e7'
branch_labels = None
depends_on = None


def upgrade():
    with op.batch_alter_table('tasks') as batch:
        batch.add_column(sa.Column('responsible_user_id', sa.Integer(), nullable=True))
        batch.add_column(sa.Column('stage_id', sa.Integer(), nullable=True))
        batch.create_foreign_key(
            'fk_tasks_responsible_user_id', 'users',
            ['responsible_user_id'], ['id'], ondelete='SET NULL'
        )
        batch.create_foreign_key(
            'fk_tasks_stage_id', 'stages',
            ['stage_id'], ['id'], ondelete='SET NULL'
        )
    op.create_index('idx_tasks_responsible', 'tasks', ['responsible_user_id'])
    op.create_index('idx_tasks_stage', 'tasks', ['stage_id'])

    op.create_table(
        'comments',
        sa.Column('id', sa.Integer(), primary_key=True),
        sa.Column('task_id', sa.Integer(), nullable=False),
        sa.Column('author_id', sa.Integer(), nullable=False),
        sa.Column('text', sa.Text(), nullable=False),
        sa.Column('created_at', sa.DateTime(timezone=True), nullable=False,
                  server_default=sa.text('NOW()')),
        sa.Column('updated_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('deleted_at', sa.DateTime(timezone=True), nullable=True),
        sa.ForeignKeyConstraint(['task_id'], ['tasks.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['author_id'], ['users.id'], ondelete='CASCADE'),
    )
    op.create_index('idx_comments_task', 'comments', ['task_id', 'created_at'])
    op.create_index('idx_comments_author', 'comments', ['author_id'])


def downgrade():
    op.drop_index('idx_comments_author', table_name='comments')
    op.drop_index('idx_comments_task', table_name='comments')
    op.drop_table('comments')

    op.drop_index('idx_tasks_stage', table_name='tasks')
    op.drop_index('idx_tasks_responsible', table_name='tasks')
    with op.batch_alter_table('tasks') as batch:
        batch.drop_constraint('fk_tasks_stage_id', type_='foreignkey')
        batch.drop_constraint('fk_tasks_responsible_user_id', type_='foreignkey')
        batch.drop_column('stage_id')
        batch.drop_column('responsible_user_id')
