"""Раздел «Мой Groove»: лента активности, реакции, комментарии, питомцы, рейды

Revision ID: a9b0c1d2e3f4
Revises: e6f7a8b9c0d1
Create Date: 2026-06-10
"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

revision = 'a9b0c1d2e3f4'
down_revision = 'e6f7a8b9c0d1'
branch_labels = None
depends_on = None


def upgrade():
    op.create_table(
        'feed_events',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('company_id', sa.Integer(), nullable=False),
        sa.Column('user_id', sa.Integer(), nullable=True),
        sa.Column('kind', sa.String(length=32), nullable=False),
        sa.Column('payload', postgresql.JSONB(astext_type=sa.Text()), nullable=False,
                  server_default=sa.text("'{}'::jsonb")),
        sa.Column('created_at', sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(['company_id'], ['companies.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['user_id'], ['users.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id'),
    )
    op.create_index('idx_feed_events_company_id', 'feed_events', ['company_id', 'id'])
    op.create_index('idx_feed_events_user', 'feed_events', ['user_id'])

    op.create_table(
        'feed_reactions',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('event_id', sa.Integer(), nullable=False),
        sa.Column('user_id', sa.Integer(), nullable=False),
        sa.Column('emoji', sa.String(length=16), nullable=False),
        sa.Column('created_at', sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(['event_id'], ['feed_events.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['user_id'], ['users.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('event_id', 'user_id', 'emoji', name='uq_feed_reaction'),
    )
    op.create_index('idx_feed_reactions_event', 'feed_reactions', ['event_id'])

    op.create_table(
        'feed_comments',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('event_id', sa.Integer(), nullable=False),
        sa.Column('author_id', sa.Integer(), nullable=True),
        sa.Column('is_bot', sa.Boolean(), nullable=False, server_default=sa.text('false')),
        sa.Column('reply_to_id', sa.Integer(), nullable=True),
        sa.Column('text', sa.Text(), nullable=False),
        sa.Column('created_at', sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(['event_id'], ['feed_events.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['author_id'], ['users.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['reply_to_id'], ['feed_comments.id'], ondelete='SET NULL'),
        sa.PrimaryKeyConstraint('id'),
    )
    op.create_index('idx_feed_comments_event', 'feed_comments', ['event_id', 'created_at'])

    op.create_table(
        'pets',
        sa.Column('user_id', sa.Integer(), nullable=False),
        sa.Column('company_id', sa.Integer(), nullable=False),
        sa.Column('name', sa.String(length=50), nullable=False),
        sa.Column('species', sa.String(length=16), nullable=False, server_default='egg'),
        sa.Column('stage', sa.Integer(), nullable=False, server_default='0'),
        sa.Column('xp', sa.Integer(), nullable=False, server_default='0'),
        sa.Column('beans', sa.Integer(), nullable=False, server_default='0'),
        sa.Column('hat', sa.String(length=32), nullable=True),
        sa.Column('accessories', postgresql.JSONB(astext_type=sa.Text()), nullable=False,
                  server_default=sa.text("'[]'::jsonb")),
        sa.Column('feed_streak', sa.Integer(), nullable=False, server_default='0'),
        sa.Column('last_fed_date', sa.Date(), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(['user_id'], ['users.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['company_id'], ['companies.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('user_id'),
    )
    op.create_index('idx_pets_company', 'pets', ['company_id'])

    op.create_table(
        'pet_strokes',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('pet_user_id', sa.Integer(), nullable=False),
        sa.Column('user_id', sa.Integer(), nullable=False),
        sa.Column('day', sa.Date(), nullable=False),
        sa.Column('created_at', sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(['pet_user_id'], ['pets.user_id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['user_id'], ['users.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('pet_user_id', 'user_id', 'day', name='uq_pet_stroke_day'),
    )
    op.create_index('idx_pet_strokes_pet_day', 'pet_strokes', ['pet_user_id', 'day'])

    op.create_table(
        'groove_raids',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('company_id', sa.Integer(), nullable=False),
        sa.Column('week_start', sa.Date(), nullable=False),
        sa.Column('boss', sa.String(length=64), nullable=False),
        sa.Column('target', sa.Integer(), nullable=False),
        sa.Column('reward', sa.String(length=32), nullable=False, server_default='helmet'),
        sa.Column('defeated_at', sa.DateTime(timezone=True), nullable=True),
        sa.Column('created_at', sa.DateTime(timezone=True), nullable=False),
        sa.ForeignKeyConstraint(['company_id'], ['companies.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id'),
        sa.UniqueConstraint('company_id', 'week_start', name='uq_raid_week'),
    )


def downgrade():
    op.drop_table('groove_raids')
    op.drop_index('idx_pet_strokes_pet_day', table_name='pet_strokes')
    op.drop_table('pet_strokes')
    op.drop_index('idx_pets_company', table_name='pets')
    op.drop_table('pets')
    op.drop_index('idx_feed_comments_event', table_name='feed_comments')
    op.drop_table('feed_comments')
    op.drop_index('idx_feed_reactions_event', table_name='feed_reactions')
    op.drop_table('feed_reactions')
    op.drop_index('idx_feed_events_user', table_name='feed_events')
    op.drop_index('idx_feed_events_company_id', table_name='feed_events')
    op.drop_table('feed_events')
