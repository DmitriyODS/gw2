from typing import Optional
from app.extensions import db
from app.models import Department

# Домен отделов живёт в tasksvc; здесь — точечное чтение для валидаций
# YouGile-импорта (до фазы 4).


def get_by_id(dept_id: int) -> Optional[Department]:
    return db.session.get(Department, dept_id)
