from app.extensions import db


class UnitType(db.Model):
    __tablename__ = "unit_types"

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(255), nullable=False, unique=True)

    units = db.relationship("Unit", back_populates="unit_type", lazy="dynamic", cascade="all, delete-orphan")
