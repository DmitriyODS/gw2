"""performance indexes for tasks, users, units and messages

Revision ID: e6f7a8b9c0d1
Revises: d4e5f6a7b8c9
Create Date: 2026-06-08 12:00:00.000000

Adds the indexes used by the hottest list and search paths:
- task filters by company + author / department / responsible / stage
- active task lookups by company + received_at
- active unit lookups by company + user
- unread message scans by conversation
- trigram search support for task names and directory search
"""
from alembic import op
import sqlalchemy as sa


revision = "e6f7a8b9c0d1"
down_revision = "d4e5f6a7b8c9"
branch_labels = None
depends_on = None


def upgrade():
    op.execute("CREATE EXTENSION IF NOT EXISTS pg_trgm")

    op.create_index("idx_tasks_company_author", "tasks", ["company_id", "author_id"])
    op.create_index("idx_tasks_company_department", "tasks", ["company_id", "department_id"])
    op.create_index("idx_tasks_company_responsible", "tasks", ["company_id", "responsible_user_id"])
    op.create_index("idx_tasks_company_stage", "tasks", ["company_id", "stage_id"])
    op.create_index(
        "idx_tasks_company_active_received", "tasks", ["company_id", "received_at"],
        unique=False, postgresql_where=sa.text("is_archived = FALSE"),
    )

    op.create_index(
        "idx_units_company_user_active", "units", ["company_id", "user_id"],
        unique=False, postgresql_where=sa.text("datetime_end IS NULL"),
    )
    op.create_index("idx_units_company_start", "units", ["company_id", "datetime_start"])

    op.create_index(
        "idx_msg_conv_unread", "messages", ["conversation_id"],
        unique=False, postgresql_where=sa.text("read_at IS NULL"),
    )

    op.create_index(
        "idx_users_company_visible", "users", ["company_id"],
        unique=False, postgresql_where=sa.text("is_hidden = FALSE"),
    )

    op.execute(
        "CREATE INDEX idx_tasks_name_trgm ON tasks USING gin (lower(name) gin_trgm_ops)"
    )
    op.execute(
        "CREATE INDEX idx_users_fio_trgm ON users USING gin (lower(fio) gin_trgm_ops)"
    )
    op.execute(
        "CREATE INDEX idx_users_login_trgm ON users USING gin (lower(login) gin_trgm_ops)"
    )


def downgrade():
    op.execute("DROP INDEX IF EXISTS idx_users_login_trgm")
    op.execute("DROP INDEX IF EXISTS idx_users_fio_trgm")
    op.execute("DROP INDEX IF EXISTS idx_tasks_name_trgm")

    op.drop_index("idx_users_company_visible", table_name="users")
    op.drop_index("idx_msg_conv_unread", table_name="messages")
    op.drop_index("idx_units_company_start", table_name="units")
    op.drop_index("idx_units_company_user_active", table_name="units")

    op.drop_index("idx_tasks_company_active_received", table_name="tasks")
    op.drop_index("idx_tasks_company_stage", table_name="tasks")
    op.drop_index("idx_tasks_company_responsible", table_name="tasks")
    op.drop_index("idx_tasks_company_department", table_name="tasks")
    op.drop_index("idx_tasks_company_author", table_name="tasks")
