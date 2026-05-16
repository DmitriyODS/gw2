from app.extensions import db


class Department(db.Model):
    __tablename__ = "departments"

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(255), nullable=False, unique=True)

    tasks = db.relationship("Task", back_populates="department", lazy="dynamic")
