from app.extensions import db
from app.repositories import company_repo, user_repo
from app.models.company import DEFAULT_SETTINGS
from app.utils.logger import get_logger

logger = get_logger(__name__)


class CompanyServiceError(Exception):
    def __init__(self, message: str, code: str = "COMPANY_ERROR", http_status: int = 400):
        self.message = message
        self.code = code
        self.http_status = http_status
        super().__init__(message)


def _merge_settings(base: dict, patch: dict | None) -> dict:
    merged = {**DEFAULT_SETTINGS, **(base or {})}
    if patch:
        merged.update(patch)
    return merged


def _validate_director(director_id: int | None, target_company_id: int | None):
    """Корневой Руководитель должен быть существующим, видимым пользователем
    и (если он уже привязан к компании) — той же компании, что и редактируется
    (для нового создания target_company_id=None — позже отдельно поправим
    привязку пользователя к компании)."""
    if director_id is None:
        return
    user = user_repo.get_by_id(director_id)
    if user is None or user.is_hidden:
        raise CompanyServiceError("Руководитель не найден", "DIRECTOR_NOT_FOUND", 404)


def create_company(name: str, description: str | None, director_id: int | None,
                   is_active: bool, settings: dict | None) -> object:
    if company_repo.get_by_name(name):
        raise CompanyServiceError("Компания с таким названием уже существует",
                                  "DUPLICATE", 409)
    _validate_director(director_id, None)

    company = company_repo.create(
        name=name, description=description, director_id=director_id,
        settings=_merge_settings({}, settings),
    )
    if not is_active:
        company_repo.update(company, is_active=False)

    # Если у выбранного директора ещё нет компании — автоматически привязываем
    # его к создаваемой. Так создание компании сразу логично «доукомплектовано».
    if director_id is not None:
        director = user_repo.get_by_id(director_id)
        if director and director.company_id is None:
            user_repo.update(director, company_id=company.id)

    db.session.commit()
    logger.info("company.create",
                extra={"extra": {"company_id": company.id, "event": "company.create"}})
    return company


def update_company(company_id: int, **kwargs) -> object:
    company = company_repo.get_by_id(company_id)
    if company is None:
        raise CompanyServiceError("Компания не найдена", "NOT_FOUND", 404)

    if "name" in kwargs and kwargs["name"] != company.name:
        existing = company_repo.get_by_name(kwargs["name"])
        if existing and existing.id != company_id:
            raise CompanyServiceError("Компания с таким названием уже существует",
                                      "DUPLICATE", 409)

    if "director_id" in kwargs:
        _validate_director(kwargs["director_id"], company_id)

    if "settings" in kwargs:
        kwargs["settings"] = _merge_settings(company.settings, kwargs["settings"])

    company_repo.update(company, **kwargs)
    db.session.commit()
    return company


def delete_company(company_id: int) -> None:
    company = company_repo.get_by_id(company_id)
    if company is None:
        raise CompanyServiceError("Компания не найдена", "NOT_FOUND", 404)

    company_repo.delete(company)
    db.session.commit()
    logger.info("company.delete",
                extra={"extra": {"company_id": company_id, "event": "company.delete"}})


def set_active(company_id: int, is_active: bool) -> object:
    company = company_repo.get_by_id(company_id)
    if company is None:
        raise CompanyServiceError("Компания не найдена", "NOT_FOUND", 404)
    company_repo.update(company, is_active=is_active)
    db.session.commit()
    return company
