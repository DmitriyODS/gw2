from app.extensions import db


class Stage(db.Model):
    __tablename__ = "stages"

    id = db.Column(db.Integer, primary_key=True)
    company_id = db.Column(db.Integer, db.ForeignKey("companies.id", ondelete="CASCADE"),
                           nullable=False)
    name = db.Column(db.String(255), nullable=False)
    # Имя цвета из палитры --tag-*: red/orange/amber/green/teal/blue/violet/pink.
    color = db.Column(db.String(16), nullable=False, server_default="blue")
    order = db.Column(db.Integer, nullable=False, server_default="0")

    company = db.relationship("Company", foreign_keys=[company_id])

    __table_args__ = (
        db.UniqueConstraint("company_id", "name", name="uq_stages_company_name"),
        db.Index("idx_stages_company_order", "company_id", "order"),
    )
