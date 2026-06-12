from app.extensions import db
from app.models import Comment

# Домен комментариев живёт в tasksvc; здесь — только системные комментарии
# YouGile-интеграции (до фазы 4).


def create(task_id: int, author_id: int, text: str) -> Comment:
    c = Comment(task_id=task_id, author_id=author_id, text=text)
    db.session.add(c)
    db.session.flush()
    return c
