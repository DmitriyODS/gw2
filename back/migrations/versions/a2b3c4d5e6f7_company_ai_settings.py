"""AI-настройки на уровне компании

Revision ID: a2b3c4d5e6f7
Revises: c6d7e8f9a0b1
Create Date: 2026-06-06 08:00:00.000000

Добавляем в companies поля, необходимые для интеграции с ProxyAPI:
- ai_enabled         — включена ли AI-функциональность (TV-факт, семантика).
- ai_api_key_enc     — Fernet-зашифрованный API-ключ (NULL пока не задан).
- ai_key_hint        — короткая маска для UI ("sk-…ab12"), хранится в открытом
                       виде специально, чтобы superadmin понимал, какой ключ
                       сейчас стоит, не расшифровывая его.
- ai_model_chat      — модель для чата/фактов (дефолт gpt-4o-mini).
- ai_model_embedding — модель эмбеддингов (дефолт text-embedding-3-small).

Никаких server_default для ключа — пустой ключ означает «AI выключен, даже
если ai_enabled=TRUE». Дефолты моделей ставим server_default'ом, чтобы
существующие компании после миграции сразу имели валидные значения.
"""
from alembic import op
import sqlalchemy as sa


revision = 'a2b3c4d5e6f7'
down_revision = 'c6d7e8f9a0b1'
branch_labels = None
depends_on = None


def upgrade():
    with op.batch_alter_table('companies') as batch:
        batch.add_column(sa.Column('ai_enabled', sa.Boolean(), nullable=False,
                                   server_default=sa.text('false')))
        batch.add_column(sa.Column('ai_api_key_enc', sa.LargeBinary(), nullable=True))
        batch.add_column(sa.Column('ai_key_hint', sa.String(length=16), nullable=True))
        batch.add_column(sa.Column('ai_model_chat', sa.String(length=64),
                                   nullable=False, server_default='gpt-4o-mini'))
        batch.add_column(sa.Column('ai_model_embedding', sa.String(length=64),
                                   nullable=False, server_default='text-embedding-3-small'))


def downgrade():
    with op.batch_alter_table('companies') as batch:
        batch.drop_column('ai_model_embedding')
        batch.drop_column('ai_model_chat')
        batch.drop_column('ai_key_hint')
        batch.drop_column('ai_api_key_enc')
        batch.drop_column('ai_enabled')
