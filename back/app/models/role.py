from app.extensions import db


class Role(db.Model):
    __tablename__ = "roles"

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(100), nullable=False, unique=True)
    level = db.Column(db.SmallInteger, nullable=False)

    users = db.relationship("User", back_populates="role", lazy="dynamic")
