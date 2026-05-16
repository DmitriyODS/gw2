from flask import Blueprint, request, jsonify, make_response
from flask_jwt_extended import jwt_required, get_jwt_identity, verify_jwt_in_request, get_jwt
from marshmallow import ValidationError

from app.schemas import LoginSchema, ChangeDefaultSchema
from app.services import auth_service
from app.services.auth_service import AuthError

bp = Blueprint("auth", __name__, url_prefix="/api/auth")

_login_schema = LoginSchema()
_change_schema = ChangeDefaultSchema()

REFRESH_COOKIE = "refresh_token"
COOKIE_MAX_AGE = 30 * 24 * 3600  # 30 дней


def _set_refresh_cookie(response, refresh_token: str):
    response.set_cookie(
        REFRESH_COOKIE,
        refresh_token,
        max_age=COOKIE_MAX_AGE,
        httponly=True,
        samesite="Strict",
        secure=False,  # True в production за nginx
    )


@bp.post("/login")
def login():
    """
    Вход в систему.
    ---
    tags: [auth]
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required: [login, password]
            properties:
              login: {type: string}
              password: {type: string}
    responses:
      200:
        description: Успешный вход
        content:
          application/json:
            schema:
              type: object
              properties:
                access_token: {type: string}
                user_id: {type: integer}
                force_change: {type: boolean}
      401:
        description: Неверные учётные данные
    """
    try:
        data = _login_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    try:
        result = auth_service.login(data["login"], data["password"])
    except AuthError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    resp = make_response(jsonify({
        "access_token": result["access_token"],
        "user_id": result["user_id"],
        "force_change": result["force_change"],
    }))
    _set_refresh_cookie(resp, result["refresh_token"])
    return resp, 200


@bp.post("/refresh")
def refresh():
    """
    Обновить access token по refresh cookie.
    ---
    tags: [auth]
    responses:
      200:
        description: Новый access token
      401:
        description: Невалидный или отсутствующий refresh token
    """
    try:
        verify_jwt_in_request(locations=["cookies"], refresh=True, cookie_name=REFRESH_COOKIE)
    except Exception:
        return jsonify({"error": "INVALID_TOKEN", "message": "Refresh token недействителен"}), 401

    user_id = get_jwt_identity()
    try:
        access_token = auth_service.refresh(user_id)
    except AuthError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    return jsonify({"access_token": access_token}), 200


@bp.post("/logout")
@jwt_required()
def logout():
    """
    Выйти из системы (очистить refresh cookie).
    ---
    tags: [auth]
    security: [BearerAuth: []]
    responses:
      200:
        description: Выход выполнен
    """
    resp = make_response(jsonify({"message": "Выход выполнен"}))
    resp.delete_cookie(REFRESH_COOKIE)
    return resp, 200


@bp.post("/change-default")
@jwt_required()
def change_default():
    """
    Сменить логин/пароль при первом входе (force_change).
    ---
    tags: [auth]
    security: [BearerAuth: []]
    requestBody:
      required: true
      content:
        application/json:
          schema:
            type: object
            required: [new_login, new_password, confirm_password]
            properties:
              new_login: {type: string, minLength: 3}
              new_password: {type: string, minLength: 8}
              confirm_password: {type: string}
    responses:
      200:
        description: Учётные данные изменены, выданы новые токены
      400:
        description: Ошибка валидации
    """
    try:
        data = _change_schema.load(request.get_json(silent=True) or {})
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400

    user_id = get_jwt_identity()
    try:
        result = auth_service.change_default_credentials(
            user_id, data["new_login"], data["new_password"], data["confirm_password"]
        )
    except AuthError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status

    resp = make_response(jsonify({"access_token": result["access_token"]}))
    _set_refresh_cookie(resp, result["refresh_token"])
    return resp, 200
