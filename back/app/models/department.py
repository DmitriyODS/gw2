from app.extensions import db


class Department(db.Model):
    __tablename__ = "departments"

    id = db.Column(db.Integer, primary_key=True)
    name = db.Column(db.String(255), nullable=False)
    company_id = db.Column(db.Integer, db.ForeignKey("companies.id", ondelete="CASCADE"),
                           nullable=False)

    company = db.relationship("Company", foreign_keys=[company_id])
    tasks = db.relationship("Task", back_populates="department", lazy="dynamic")

    __table_args__ = (
        db.UniqueConstraint("company_id", "name", name="uq_departments_company_name"),
        db.Index("idx_departments_company", "company_id"),
    )
