from datetime import datetime, date, timezone
from flask import Blueprint, request, jsonify, send_file
from flask_jwt_extended import get_jwt_identity

from app.services import stats_service
from app.repositories import user_repo
from app.utils.permissions import (
    require_role, require_auth, EMPLOYEE, MANAGER,
    get_user_level, resolve_company_scope,
)
from flask import g

bp = Blueprint("stats", __name__, url_prefix="/api/stats")

_DEFAULT_FROM = datetime(date.today().year, 1, 1, tzinfo=timezone.utc)
_DEFAULT_TO = datetime(date.today().year, 12, 31, 23, 59, 59, tzinfo=timezone.utc)


def _parse_period(args) -> tuple[datetime, datetime]:
    try:
        from_str = args.get("from")
        to_str = args.get("to")
        period_start = datetime.fromisoformat(from_str) if from_str else _DEFAULT_FROM
        period_end = datetime.fromisoformat(to_str) if to_str else _DEFAULT_TO
        if period_start.tzinfo is None:
            period_start = period_start.replace(tzinfo=timezone.utc)
        if period_end.tzinfo is None:
            period_end = period_end.replace(tzinfo=timezone.utc)
        # date-only string → extend to end of day so the full day is included
        if to_str and "T" not in to_str:
            period_end = period_end.replace(hour=23, minute=59, second=59, microsecond=999999)
        return period_start, period_end
    except ValueError:
        raise ValueError("Неверный формат даты. Используйте YYYY-MM-DD")


@bp.get("/common")
@require_role(EMPLOYEE)
def get_common():
    """
    Общая статистика.
    ---
    tags: [stats]
    security: [BearerAuth: []]
    parameters:
      - {in: query, name: from, schema: {type: string, format: date}, description: Начало периода}
      - {in: query, name: to, schema: {type: string, format: date}, description: Конец периода}
    responses:
      200:
        description: Общая статистика
    """
    try:
        period_start, period_end = _parse_period(request.args)
    except ValueError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": str(e)}), 400

    company_id = resolve_company_scope(g.current_user)
    data = stats_service.get_common(period_start, period_end, company_id)
    return jsonify(data), 200


@bp.get("/extended")
@require_role(EMPLOYEE)
def get_extended():
    """
    Расширенная статистика.
    ---
    tags: [stats]
    security: [BearerAuth: []]
    parameters:
      - {in: query, name: from, schema: {type: string, format: date}}
      - {in: query, name: to, schema: {type: string, format: date}}
    responses:
      200:
        description: Расширенная статистика
    """
    try:
        period_start, period_end = _parse_period(request.args)
    except ValueError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": str(e)}), 400

    company_id = resolve_company_scope(g.current_user)
    data = stats_service.get_extended(period_start, period_end, company_id)
    return jsonify(data), 200


@bp.get("/common/export")
@require_role(MANAGER)
def export_common():
    """
    Выгрузить общую статистику в XLSX.
    ---
    tags: [stats]
    security: [BearerAuth: []]
    parameters:
      - {in: query, name: from, schema: {type: string, format: date}}
      - {in: query, name: to, schema: {type: string, format: date}}
    responses:
      200:
        description: XLSX-файл
        content:
          application/vnd.openxmlformats-officedocument.spreadsheetml.sheet: {}
    """
    try:
        period_start, period_end = _parse_period(request.args)
    except ValueError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": str(e)}), 400

    company_id = resolve_company_scope(g.current_user)
    buf = stats_service.export_common_xlsx(period_start, period_end, company_id)
    return send_file(
        buf,
        mimetype="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        as_attachment=True,
        download_name=f"stats_common_{period_start.date()}_{period_end.date()}.xlsx",
    )


@bp.get("/extended/export")
@require_role(MANAGER)
def export_extended():
    """
    Выгрузить расширенную статистику в XLSX.
    ---
    tags: [stats]
    security: [BearerAuth: []]
    parameters:
      - {in: query, name: from, schema: {type: string, format: date}}
      - {in: query, name: to, schema: {type: string, format: date}}
    responses:
      200:
        description: XLSX-файл
        content:
          application/vnd.openxmlformats-officedocument.spreadsheetml.sheet: {}
    """
    try:
        period_start, period_end = _parse_period(request.args)
    except ValueError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": str(e)}), 400

    company_id = resolve_company_scope(g.current_user)
    buf = stats_service.export_extended_xlsx(period_start, period_end, company_id)
    return send_file(
        buf,
        mimetype="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        as_attachment=True,
        download_name=f"stats_extended_{period_start.date()}_{period_end.date()}.xlsx",
    )


@bp.get("/user-tasks")
@require_role(EMPLOYEE)
def get_user_tasks():
    """
    Задачи с участием сотрудника за период.
    ---
    tags: [stats]
    security: [BearerAuth: []]
    parameters:
      - {in: query, name: user_id, schema: {type: integer}, description: ID пользователя (менеджер+ может запросить любого)}
      - {in: query, name: from, schema: {type: string, format: date}}
      - {in: query, name: to, schema: {type: string, format: date}}
    responses:
      200:
        description: Список задач с суммарным временем
    """
    try:
        period_start, period_end = _parse_period(request.args)
    except ValueError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": str(e)}), 400

    current_user_id = int(get_jwt_identity())

    user_id_str = request.args.get("user_id")
    if user_id_str:
        requested_user_id = int(user_id_str)
        if requested_user_id != current_user_id:
            current_user = user_repo.get_by_id(current_user_id)
            if get_user_level(current_user) < MANAGER:
                return jsonify({"error": "FORBIDDEN", "message": "Доступ запрещён"}), 403
            target = user_repo.get_by_id(requested_user_id)
            if target is None:
                return jsonify({"error": "NOT_FOUND", "message": "Сотрудник не найден"}), 404
            # Менеджер/Руководитель могут смотреть только в своей компании.
            # Администратор системы (company_id=None) — кого угодно.
            if current_user.company_id is not None and target.company_id != current_user.company_id:
                return jsonify({"error": "FORBIDDEN", "message": "Доступ запрещён"}), 403
        target_user_id = requested_user_id
    else:
        target_user_id = current_user_id

    data = stats_service.get_user_tasks(target_user_id, period_start, period_end)
    return jsonify(data), 200


@bp.get("/employees")
@require_role(MANAGER)
def get_employees():
    """
    Список сотрудников для выбора в статистике.
    ---
    tags: [stats]
    security: [BearerAuth: []]
    responses:
      200:
        description: Список пользователей (id, fio)
    """
    company_id = resolve_company_scope(g.current_user)
    users = user_repo.get_all(include_hidden=False, company_id=company_id)
    return jsonify([{"id": u.id, "fio": u.fio} for u in users]), 200


@bp.get("/responsibles")
@require_role(EMPLOYEE)
def get_responsibles():
    """
    Сотрудники-ответственные — список с количеством открытых/закрытых задач.
    ---
    tags: [stats]
    security: [BearerAuth: []]
    parameters:
      - {in: query, name: company_id, schema: {type: integer}, required: false}
    responses:
      200:
        description: Список ответственных
    """
    company_id = resolve_company_scope(g.current_user)
    data = stats_service.get_responsibles(company_id)
    return jsonify(data), 200


@bp.get("/profile")
@require_auth
def get_profile():
    """
    Личная статистика текущего пользователя.
    ---
    tags: [stats]
    security: [BearerAuth: []]
    parameters:
      - {in: query, name: from, schema: {type: string, format: date}}
      - {in: query, name: to, schema: {type: string, format: date}}
    responses:
      200:
        description: Личная статистика
    """
    try:
        period_start, period_end = _parse_period(request.args)
    except ValueError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": str(e)}), 400

    user_id = int(get_jwt_identity())
    data = stats_service.get_profile(user_id, period_start, period_end)
    return jsonify(data), 200
