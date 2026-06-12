from typing import Optional
from app.extensions import db
from app.models import Company

# CRUD компаний живёт в authsvc (back-go/auth); здесь — только точечные
# чтения для оставшихся Flask-доменов (флаги settings в задачах).


def get_by_id(company_id: int) -> Optional[Company]:
    return db.session.get(Company, company_id)
