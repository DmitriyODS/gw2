from app.extensions import db


class UnitType(db.Model):
    __tablename__ = "unit_types"

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(255), nullable=False)
    company_id = db.Column(db.Integer, db.ForeignKey("companies.id", ondelete="CASCADE"),
                           nullable=False)

    company = db.relationship("Company", foreign_keys=[company_id])
    units = db.relationship("Unit", back_populates="unit_type", lazy="dynamic", cascade="all, delete-orphan")

    __table_args__ = (
        db.UniqueConstraint("company_id", "name", name="uq_unit_types_company_name"),
        db.Index("idx_unit_types_company", "company_id"),
    )
