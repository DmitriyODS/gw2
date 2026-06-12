"""Эндпоинты подключения к YouGile.

Структура:
- Личный коннект пользователя (/account/...) — любой авторизованный.
- Админ-визард компании (/projects, /boards, /columns, /settings) — DIRECTOR+.

Импорт/экспорт задач и webhook-ingress появятся на этапах 3–4.
"""
from flask import Blueprint, g, jsonify, request
from marshmallow import ValidationError

from app.extensions import db, socketio
from app.integrations.yougile.account_service import (
    YougileAccountError, connect_for_user, disconnect, get_status,
    list_companies_for_credentials, rotate,
)
from app.integrations.yougile.company_service import (
    YougileCompanyError, get_settings, list_boards, list_columns,
    list_projects, reset_integration, update_settings,
)
from app.integrations.yougile.crypto import YougileSecretMisconfigured
from app.integrations.yougile.task_service import (
    ImportPayload, YougileTaskError, export_to_yougile, import_from_url,
    unlink_task,
)
from app.integrations.yougile.task_apply import apply_event
from app.integrations.yougile.webhook_service import (
    YougileWebhookError, verify_secret,
)
from app.models.company import Company
from app.repositories import task_repo
from app.schemas.task import TaskSchema
from app.schemas.yougile import (
    YougileAccountStatusSchema, YougileBoardSchema, YougileColumnSchema,
    YougileCompanyListItemSchema, YougileCompanySettingsSchema,
    YougileCompanySettingsUpdateSchema, YougileConnectFinishSchema,
    YougileConnectStartSchema, YougileExportTaskSchema, YougileImportTaskSchema,
    YougileProjectSchema, YougileRotateSchema,
)
from app.utils.logger import get_logger
from app.utils.permissions import DIRECTOR, EMPLOYEE, require_auth, require_role


bp = Blueprint("yougile", __name__, url_prefix="/api/yougile")
logger = get_logger(__name__)


_status_schema = YougileAccountStatusSchema()
_company_item_schema = YougileCompanyListItemSchema(many=True)
_project_schema = YougileProjectSchema(many=True)
_board_schema = YougileBoardSchema(many=True)
_column_schema = YougileColumnSchema(many=True)
_settings_schema = YougileCompanySettingsSchema()
_settings_update_schema = YougileCompanySettingsUpdateSchema()
_import_schema = YougileImportTaskSchema()
_export_schema = YougileExportTaskSchema()
_task_schema = TaskSchema()


def _err_task(e: YougileTaskError):
    return jsonify({"error": e.code, "message": e.message}), e.http_status


def _dump_task(task, current_user_id: int) -> dict:
    """Полный dump задачи в том же виде, что отдаёт /api/tasks (with favorite,
    has_units, user-color и т.п.). Дёргаем _enrich_task лениво, чтобы не было
    кругового импорта на верхнем уровне модуля."""
    from app.integrations.yougile.task_dump import enrich_task as _enrich_task  # noqa: WPS433
    return _enrich_task(task, current_user_id)


# ── обработчики ошибок ────────────────────────────────────────────────────

def _err_account(e: YougileAccountError):
    return jsonify({"error": e.code, "message": e.message}), 400


def _err_company(e: YougileCompanyError):
    return jsonify({"error": e.code, "message": e.message}), 400


def _dump_settings(s) -> dict:
    """Сериализация CompanyYougileSettings в ответ API (общая для get/put/reset)."""
    return _settings_schema.dump({
        "enabled": s.enabled,
        "yg_company_id": s.yg_company_id,
        "yg_company_name": s.yg_company_name,
        "yg_project_id": s.yg_project_id,
        "yg_project_title": s.yg_project_title,
        "yg_board_id": s.yg_board_id,
        "yg_board_title": s.yg_board_title,
        "yg_first_column_id": s.yg_first_column_id,
        "yg_completed_column_id": s.yg_completed_column_id,
        "webhook_registered": s.webhook_registered,
    })


def _err_misconfig(e: YougileSecretMisconfigured):
    # Это 500: на сервере не задан YOUGILE_ENC_KEY. Видно админу — он
    # знает, что нужно поправить env.
    return jsonify({
        "error": "ENC_KEY_MISCONFIGURED",
        "message": "На сервере не задан YOUGILE_ENC_KEY",
    }), 500


# ── статус и личное подключение ───────────────────────────────────────────

@bp.get("/status")
@require_auth
def status():
    s = get_status(g.current_user)
    return jsonify(_status_schema.dump({
        "connected": s.connected,
        "yg_login": s.yg_login,
        "key_fingerprint": s.key_fingerprint,
        "last_validated_at": s.last_validated_at,
        "yg_company_id": s.yg_company_id,
        "company_enabled": s.company_enabled,
    })), 200


@bp.post("/account")
@require_auth
def connect_account():
    """Подключение обычным пользователем — yg_company_id берётся из компании.

    Если в payload передан yg_company_id, он используется только когда
    пользователь — DIRECTOR+ (админ выбирает свою же будущую компанию ещё
    до сохранения настроек).
    """
    try:
        data = YougileConnectFinishSchema().load(request.get_json() or {})
    except ValidationError as ve:
        return jsonify({"error": "VALIDATION", "details": ve.messages}), 400
    try:
        user = g.current_user
        explicit_id = data.get("yg_company_id")
        # Обычный юзер не может выбрать произвольную yg_company — у него
        # она зафиксирована настройками; молча игнорируем поле.
        is_admin = user.role and user.role.level >= DIRECTOR
        acc = connect_for_user(
            user, data["login"], data["password"],
            explicit_yg_company_id=explicit_id if is_admin else None,
        )
    except YougileAccountError as e:
        return _err_account(e)
    except YougileSecretMisconfigured as e:
        return _err_misconfig(e)
    return jsonify({"connected": True,
                    "yg_login": acc.yg_login,
                    "key_fingerprint": acc.key_fingerprint,
                    "yg_company_id": acc.yg_company_id}), 200


@bp.delete("/account")
@require_auth
def disconnect_account():
    try:
        disconnect(g.current_user)
    except YougileSecretMisconfigured as e:
        return _err_misconfig(e)
    return jsonify({"connected": False}), 200


@bp.post("/account/rotate")
@require_auth
def rotate_account():
    try:
        data = YougileRotateSchema().load(request.get_json() or {})
    except ValidationError as ve:
        return jsonify({"error": "VALIDATION", "details": ve.messages}), 400
    try:
        acc = rotate(g.current_user, data["password"])
    except YougileAccountError as e:
        return _err_account(e)
    except YougileSecretMisconfigured as e:
        return _err_misconfig(e)
    return jsonify({"connected": True,
                    "key_fingerprint": acc.key_fingerprint}), 200


# ── админ-визард: каталоги YG ─────────────────────────────────────────────

@bp.post("/companies/lookup")
@require_role(DIRECTOR)
def list_yg_companies():
    """`POST /auth/companies` под капотом — отдаём админу выбор."""
    try:
        data = YougileConnectStartSchema().load(request.get_json() or {})
    except ValidationError as ve:
        return jsonify({"error": "VALIDATION", "details": ve.messages}), 400
    try:
        items = list_companies_for_credentials(data["login"], data["password"])
    except YougileAccountError as e:
        return _err_account(e)
    return jsonify(_company_item_schema.dump(items)), 200


@bp.get("/projects")
@require_role(DIRECTOR)
def projects():
    try:
        items = list_projects(g.current_user)
    except YougileCompanyError as e:
        return _err_company(e)
    except YougileSecretMisconfigured as e:
        return _err_misconfig(e)
    return jsonify(_project_schema.dump(items)), 200


@bp.get("/boards")
@require_role(DIRECTOR)
def boards():
    project_id = (request.args.get("projectId") or "").strip()
    if not project_id:
        return jsonify({"error": "VALIDATION",
                        "message": "Нужен параметр projectId"}), 400
    try:
        items = list_boards(g.current_user, project_id)
    except YougileCompanyError as e:
        return _err_company(e)
    except YougileSecretMisconfigured as e:
        return _err_misconfig(e)
    return jsonify(_board_schema.dump(items)), 200


@bp.get("/columns")
@require_role(DIRECTOR)
def columns():
    board_id = (request.args.get("boardId") or "").strip()
    if not board_id:
        return jsonify({"error": "VALIDATION",
                        "message": "Нужен параметр boardId"}), 400
    try:
        items = list_columns(g.current_user, board_id)
    except YougileCompanyError as e:
        return _err_company(e)
    except YougileSecretMisconfigured as e:
        return _err_misconfig(e)
    return jsonify(_column_schema.dump(items)), 200


# ── настройки компании ───────────────────────────────────────────────────

def _own_company_or_403() -> Company | tuple:
    """Возвращает Company пользователя или 400-tuple для эндпоинтов, которым
    компания нужна обязательно (например, save settings)."""
    user = g.current_user
    company = user.company
    if company is None:
        return jsonify({"error": "NO_COMPANY"}), 400
    return company


def _empty_settings_payload() -> dict:
    """Дефолтный ответ /company-settings: «интеграция не настроена».

    Возвращается, когда у пользователя нет компании (root admin до выбора
    компании) — чтобы фронт мог отрендерить визард, а не ловить 400.
    """
    return {
        "enabled": False,
        "yg_company_id": None,
        "yg_company_name": None,
        "yg_project_id": None,
        "yg_project_title": None,
        "yg_board_id": None,
        "yg_board_title": None,
        "yg_first_column_id": None,
        "yg_completed_column_id": None,
        "webhook_registered": False,
    }


@bp.get("/company-settings")
@require_role(DIRECTOR)
def get_company_settings():
    user = g.current_user
    company = user.company
    if company is None:
        # Root admin без выбранной компании — отдаём пустые настройки, чтобы
        # фронт мог нормально открыть страницу и предложить настройку.
        return jsonify(_settings_schema.dump(_empty_settings_payload())), 200
    s = get_settings(company)
    return jsonify(_dump_settings(s)), 200


@bp.put("/company-settings")
@require_role(DIRECTOR)
def put_company_settings():
    company = _own_company_or_403()
    if isinstance(company, tuple):
        return company
    try:
        payload = _settings_update_schema.load(request.get_json() or {})
    except ValidationError as ve:
        return jsonify({"error": "VALIDATION", "details": ve.messages}), 400
    try:
        s = update_settings(g.current_user, company, payload)
    except YougileCompanyError as e:
        return _err_company(e)
    except YougileSecretMisconfigured as e:
        return _err_misconfig(e)
    return jsonify(_dump_settings(s)), 200


@bp.post("/reset")
@require_role(DIRECTOR)
def reset_company_integration():
    """Полный сброс интеграции «начать заново» (руководитель компании).

    Снимает webhook, чистит конфигурацию компании и отвязывает личный
    YouGile-аккаунт инициатора. После — визард открывается с чистого листа.
    """
    company = _own_company_or_403()
    if isinstance(company, tuple):
        return company
    try:
        s = reset_integration(g.current_user, company)
    except YougileSecretMisconfigured as e:
        return _err_misconfig(e)
    return jsonify(_dump_settings(s)), 200


# ── импорт / экспорт / отвязка задачи ─────────────────────────────────────

@bp.post("/import-task")
@require_role(EMPLOYEE)
def import_task():
    """Создать в Groove Work задачу по ссылке на карточку YouGile.

    Пользователь должен сам выбрать отдел (как при обычном создании).
    После успешного импорта в YG-чат карточки публикуется сообщение со
    ссылкой на GW-задачу, а в GW-чате — системный комментарий со ссылкой
    на YG.
    """
    try:
        data = _import_schema.load(request.get_json() or {})
    except ValidationError as ve:
        return jsonify({"error": "VALIDATION", "details": ve.messages}), 400
    try:
        task = import_from_url(
            g.current_user,
            ImportPayload(
                url=data["url"],
                department_id=data["department_id"],
                responsible_user_id=data.get("responsible_user_id"),
                stage_id=data.get("stage_id"),
                pull_deadline=data.get("pull_deadline", True),
            ),
            origin=request.url_root,
        )
    except YougileTaskError as e:
        return _err_task(e)
    except YougileSecretMisconfigured as e:
        return _err_misconfig(e)

    payload = _dump_task(task, g.current_user.id)
    socketio.emit("task:created", payload, room="all")
    return jsonify(payload), 201


@bp.post("/export-task")
@require_role(EMPLOYEE)
def export_task():
    """Создать карточку в YouGile из существующей GW-задачи и связать их."""
    try:
        data = _export_schema.load(request.get_json() or {})
    except ValidationError as ve:
        return jsonify({"error": "VALIDATION", "details": ve.messages}), 400
    try:
        task = export_to_yougile(g.current_user, data["gw_task_id"],
                                 origin=request.url_root)
    except YougileTaskError as e:
        return _err_task(e)
    except YougileSecretMisconfigured as e:
        return _err_misconfig(e)

    payload = _dump_task(task, g.current_user.id)
    socketio.emit("task:updated", payload, room="all")
    return jsonify(payload), 200


@bp.delete("/tasks/<int:gw_task_id>/link")
@require_role(EMPLOYEE)
def unlink_yougile_task(gw_task_id: int):
    """Разорвать связь GW-задачи с YouGile (карточка в YG не удаляется)."""
    try:
        task = unlink_task(g.current_user, gw_task_id)
    except YougileTaskError as e:
        return _err_task(e)
    payload = _dump_task(task, g.current_user.id)
    socketio.emit("task:updated", payload, room="all")
    return jsonify(payload), 200


# ── webhook ingress (публичный, без токена) ──────────────────────────────────

@bp.post("/webhook/<int:company_id>/<string:secret>")
def webhook_ingress(company_id: int, secret: str):
    """Приём событий YouGile.

    Без JWT — авторизация через secret в URL. Возвращаем 2xx, чтобы YG не
    ставил retry-задержку даже если задача уже неактуальна; реальная судьба
    события — в JSON-ответе и логах.
    """
    company = db.session.get(Company, company_id)
    if company is None or not verify_secret(company, secret):
        # Не светим деталями — просто 404 для злоумышленника.
        return jsonify({"error": "NOT_FOUND"}), 404

    payload = request.get_json(silent=True) or {}
    # YouGile может слать одиночное событие или массив — приводим к списку.
    events = payload if isinstance(payload, list) else [payload]
    results = []
    for ev in events:
        try:
            results.append(apply_event(company, ev))
        except Exception as e:  # noqa: BLE001
            # Один сбойный event не должен ронять весь batch — логируем и
            # отвечаем 200, иначе YG переотправит ВЕСЬ batch. Обязательный
            # rollback: иначе грязная сессия после сбоя отравит commit'ы
            # следующих событий батча.
            db.session.rollback()
            logger.exception("yougile.webhook_apply_failed",
                             extra={"company_id": company_id,
                                    "event": ev.get("event") if isinstance(ev, dict) else None})
            results.append({"status": "error", "message": str(e)})
    return jsonify({"results": results}), 200


# ── ручная регистрация webhook'а (на случай «сбросилось»/«поменяли URL») ──

@bp.post("/webhook/register")
@require_role(DIRECTOR)
def register_webhook():
    company = _own_company_or_403()
    if isinstance(company, tuple):
        return company
    from app.integrations.yougile.webhook_service import ensure_registered
    try:
        ensure_registered(g.current_user, company)
    except YougileWebhookError as e:
        return jsonify({"error": e.code, "message": e.message}), 400
    return jsonify({"webhook_registered": bool(company.yg_webhook_id)}), 200
