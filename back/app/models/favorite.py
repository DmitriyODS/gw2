from app.extensions import db


class Favorite(db.Model):
    __tablename__ = "favorites"

    task_id = db.Column(db.Integer, db.ForeignKey("tasks.id", ondelete="CASCADE"), primary_key=True)
    user_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"), primary_key=True)

    task = db.relationship("Task", back_populates="favorites")
    user = db.relationship("User", back_populates="favorites")
