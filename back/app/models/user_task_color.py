from app.extensions import db


class UserTaskColor(db.Model):
    """Индивидуальный цвет карточки задачи на пользователя.

    Одна и та же задача может выглядеть у разных пользователей разным цветом
    (или вовсе без цвета — для этого запись просто удаляется).
    """
    __tablename__ = "user_task_colors"

    user_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"), primary_key=True)
    task_id = db.Column(db.Integer, db.ForeignKey("tasks.id", ondelete="CASCADE"), primary_key=True)
    color = db.Column(db.String(20), nullable=False)
