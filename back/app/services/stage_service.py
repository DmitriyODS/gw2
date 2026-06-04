from app.extensions import db
from app.repositories import stage_repo


class StageServiceError(Exception):
    def __init__(self, message: str, code: str = "STAGE_ERROR", http_status: int = 400):
        self.message = message
        self.code = code
        self.http_status = http_status
        super().__init__(message)


def list_stages(company_id: int):
    return stage_repo.get_all(company_id)


def create_stage(company_id: int, name: str, color: str):
    if stage_repo.get_by_name(name, company_id):
        raise StageServiceError("Этап с таким именем уже существует", "DUPLICATE", 409)
    stage = stage_repo.create(name, color, company_id, stage_repo.next_order(company_id))
    db.session.commit()
    return stage


def update_stage(company_id: int, stage_id: int, **kwargs):
    stage = stage_repo.get_by_id(stage_id)
    if stage is None or stage.company_id != company_id:
        raise StageServiceError("Этап не найден", "NOT_FOUND", 404)
    if "name" in kwargs and kwargs["name"] != stage.name:
        existing = stage_repo.get_by_name(kwargs["name"], company_id)
        if existing and existing.id != stage_id:
            raise StageServiceError("Этап с таким именем уже существует",
                                    "DUPLICATE", 409)
    stage = stage_repo.update(stage, **kwargs)
    db.session.commit()
    return stage


def delete_stage(company_id: int, stage_id: int) -> None:
    stage = stage_repo.get_by_id(stage_id)
    if stage is None or stage.company_id != company_id:
        raise StageServiceError("Этап не найден", "NOT_FOUND", 404)
    stage_repo.delete(stage)
    db.session.commit()


def reorder_stages(company_id: int, ordered_ids: list[int]):
    stages = stage_repo.reorder(company_id, ordered_ids)
    db.session.commit()
    return stages
