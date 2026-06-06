"""Эмбеддинги задач для семантического поиска

Revision ID: b3c4d5e6f7a8
Revises: a2b3c4d5e6f7
Create Date: 2026-06-06 09:00:00.000000

Расширение `vector` (pgvector) + таблица `task_embeddings`. На каждую задачу —
одна строка с вектором её текста (название + отдел + ответственный) и пометкой,
какой моделью эмбеддинг получен. При смене модели в настройках компании
получаем строки с другим `model` и фильтруем их при поиске — пока бэкфилл не
пересчитает заново.

HNSW индекс по cosine — подходит и для маленьких компаний (< 1k задач) и для
растущих, без необходимости тюнинга списков как в IVFFlat.
"""
from alembic import op
import sqlalchemy as sa


revision = 'b3c4d5e6f7a8'
down_revision = 'a2b3c4d5e6f7'
branch_labels = None
depends_on = None


# Размерность под `text-embedding-3-small` (дефолт). Если когда-нибудь
# понадобится `-large` (3072) — потребуется новая колонка/таблица. Менять in
# place нельзя: HNSW индекс привязан к размерности.
EMBEDDING_DIM = 1536


def upgrade():
    # Pgvector. На образе pgvector/pgvector:pg16 расширение есть в наборе.
    op.execute("CREATE EXTENSION IF NOT EXISTS vector")

    op.create_table(
        'task_embeddings',
        sa.Column('task_id', sa.Integer(), nullable=False),
        sa.Column('company_id', sa.Integer(), nullable=False),
        sa.Column('embedding', sa.dialects.postgresql.ARRAY(sa.Float),
                  nullable=False),  # placeholder, переопределим как vector ниже
        sa.Column('model', sa.String(length=64), nullable=False),
        sa.Column('updated_at', sa.DateTime(timezone=True),
                  nullable=False, server_default=sa.text('now()')),
        sa.PrimaryKeyConstraint('task_id'),
        sa.ForeignKeyConstraint(['task_id'], ['tasks.id'], ondelete='CASCADE'),
        sa.ForeignKeyConstraint(['company_id'], ['companies.id'], ondelete='CASCADE'),
    )

    # alembic не умеет сразу нарисовать тип `vector(N)`, поэтому колонку
    # перетипизируем сырым SQL — и только теперь можно повесить HNSW индекс.
    op.execute(f"ALTER TABLE task_embeddings "
               f"ALTER COLUMN embedding TYPE vector({EMBEDDING_DIM}) "
               f"USING embedding::vector({EMBEDDING_DIM})")

    op.create_index('idx_task_emb_company', 'task_embeddings', ['company_id'])
    op.execute(
        "CREATE INDEX idx_task_emb_hnsw ON task_embeddings "
        "USING hnsw (embedding vector_cosine_ops)"
    )


def downgrade():
    op.drop_index('idx_task_emb_hnsw', table_name='task_embeddings')
    op.drop_index('idx_task_emb_company', table_name='task_embeddings')
    op.drop_table('task_embeddings')
    # Расширение vector не дропаем — оно может пригодиться другим фичам.
