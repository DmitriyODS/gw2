"""pet species shop: unlocked_species + daily quest fields

Revision ID: c0d1e2f3a4b5
Revises: b0c1d2e3f4a5
Create Date: 2026-06-10 12:00:00
"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import JSONB

revision = "c0d1e2f3a4b5"
down_revision = "b0c1d2e3f4a5"
branch_labels = None
depends_on = None


def upgrade():
    with op.batch_alter_table("pets") as batch:
        batch.add_column(sa.Column(
            "unlocked_species", JSONB, nullable=False, server_default=sa.text("'[]'::jsonb")))
        batch.add_column(sa.Column("quest_date", sa.Date(), nullable=True))
        batch.add_column(sa.Column("quest_kind", sa.String(32), nullable=True))
        batch.add_column(sa.Column("quest_target", sa.Integer(), nullable=True))
        batch.add_column(sa.Column("quest_progress", sa.Integer(),
                                   nullable=False, server_default="0"))
        batch.add_column(sa.Column("quest_claimed", sa.Boolean(),
                                   nullable=False, server_default=sa.text("false")))


def downgrade():
    with op.batch_alter_table("pets") as batch:
        batch.drop_column("quest_claimed")
        batch.drop_column("quest_progress")
        batch.drop_column("quest_target")
        batch.drop_column("quest_kind")
        batch.drop_column("quest_date")
        batch.drop_column("unlocked_species")
