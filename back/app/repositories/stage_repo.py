from typing import Optional
from app.extensions import db
from app.models import Stage

# Домен этапов живёт в tasksvc; здесь — точечное чтение для валидаций
# YouGile-импорта (до фазы 4).


def get_by_id(stage_id: int) -> Optional[Stage]:
    return db.session.get(Stage, stage_id)
