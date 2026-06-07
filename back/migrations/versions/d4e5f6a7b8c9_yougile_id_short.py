"""tasks.yougile_id_short — короткий человекочитаемый id карточки

Revision ID: d4e5f6a7b8c9
Revises: c3d4e5f6a7b8
Create Date: 2026-06-07 17:00:00.000000

YouGile хранит у каждой карточки человекочитаемый id вида `OIP1-2454`
(поле `idTaskProject`). Он используется в коротких ссылках
`yougile.com/team/<shortTeamId>/#OIP1-2454`, которые пользователь видит
в адресной строке. Без сохранённого idTaskProject мы не можем ни принять
такую ссылку на импорте, ни сгенерировать «правильную» ссылку обратно —
поэтому добавляем колонку и заполняем при импорте/экспорте.
"""
from alembic import op
import sqlalchemy as sa


revision = 'd4e5f6a7b8c9'
down_revision = 'c3d4e5f6a7b8'
branch_labels = None
depends_on = None


def upgrade():
    with op.batch_alter_table('tasks') as batch:
        batch.add_column(sa.Column('yougile_id_short', sa.String(length=64), nullable=True))


def downgrade():
    with op.batch_alter_table('tasks') as batch:
        batch.drop_column('yougile_id_short')
