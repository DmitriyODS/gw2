"""Раздел «Мой Groove»: лента, реакции, комментарии, кудосы, заряды,
питомцы-Грувики, зоопарк, магазин и недельный рейд."""
from flask import Blueprint, g, jsonify, request
from marshmallow import ValidationError

from app.schemas.groove import (FeedReactionToggleSchema, FeedCommentCreateSchema,
                                KudosSchema, ZapSchema, PetRenameSchema,
                                ShopBuySchema, PetEquipSchema, FEED_REACTIONS)
from app.services import feed_service, pet_service
from app.services.feed_service import FeedServiceError
from app.services.pet_service import PetServiceError, SHOP_PRICES
from app.utils.permissions import require_auth, require_company_scope, get_user_level

bp = Blueprint("groove", __name__, url_prefix="/api/groove")

_reaction_schema = FeedReactionToggleSchema()
_comment_create_schema = FeedCommentCreateSchema()
_kudos_schema = KudosSchema()
_zap_schema = ZapSchema()
_rename_schema = PetRenameSchema()
_buy_schema = ShopBuySchema()
_equip_schema = PetEquipSchema()


def _load(schema):
    return schema.load(request.get_json(silent=True) or {})


# ───────────────────────────── лента ───────────────────────────────

@bp.get("/feed")
@require_auth
@require_company_scope
def get_feed():
    """
    Лента «Мой Groove» (курсорная пагинация по before_id).
    ---
    tags: [groove]
    security: [BearerAuth: []]
    parameters:
      - {in: query, name: before_id, schema: {type: integer}}
      - {in: query, name: limit, schema: {type: integer}}
    responses:
      200: {description: Страница ленты}
    """
    before_id = request.args.get("before_id", type=int)
    limit = request.args.get("limit", type=int)
    page = feed_service.get_feed_page(g.company_id, g.current_user.id,
                                      before_id, limit or feed_service.FEED_PAGE_LIMIT)
    page["allowed_reactions"] = list(FEED_REACTIONS)
    return jsonify(page), 200


@bp.post("/feed/<int:event_id>/reactions")
@require_auth
@require_company_scope
def toggle_reaction(event_id: int):
    """
    Поставить/снять реакцию на событие ленты.
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Текущее состояние реакции}
    """
    try:
        data = _load(_reaction_schema)
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    try:
        result = feed_service.toggle_reaction(event_id, g.current_user.id,
                                              g.company_id, data["emoji"])
    except FeedServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(result), 200


@bp.get("/feed/<int:event_id>/comments")
@require_auth
@require_company_scope
def list_comments(event_id: int):
    """
    Комментарии события ленты.
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Список комментариев}
    """
    try:
        items = feed_service.list_comments(event_id, g.company_id)
    except FeedServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(items), 200


@bp.post("/feed/<int:event_id>/comments")
@require_auth
@require_company_scope
def add_comment(event_id: int):
    """
    Добавить комментарий (опционально — ответ на другой комментарий).
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      201: {description: Созданный комментарий}
    """
    try:
        data = _load(_comment_create_schema)
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    try:
        comment = feed_service.add_comment(event_id, g.current_user.id,
                                           g.company_id, data["text"],
                                           data.get("reply_to_id"))
    except FeedServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(comment), 201


@bp.delete("/comments/<int:comment_id>")
@require_auth
@require_company_scope
def delete_comment(comment_id: int):
    """
    Удалить комментарий (автор или директор+).
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Комментарий удалён}
    """
    try:
        feed_service.delete_comment(comment_id, g.current_user.id,
                                    get_user_level(g.current_user))
    except FeedServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify({"message": "Комментарий удалён"}), 200


# ─────────────────────── кудосы, live, заряды ──────────────────────

@bp.post("/kudos")
@require_auth
@require_company_scope
def send_kudos():
    """
    Публичная благодарность коллеге (событие ленты).
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      201: {description: Кудос отправлен}
    """
    try:
        data = _load(_kudos_schema)
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    try:
        feed_service.send_kudos(g.company_id, g.current_user.id,
                                data["to_user_id"], data["text"])
    except FeedServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify({"message": "Кудос отправлен"}), 201


@bp.get("/live")
@require_auth
@require_company_scope
def get_live():
    """
    «Сейчас в эфире» — активные юниты компании с зарядами.
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Список активных юнитов}
    """
    return jsonify(feed_service.get_live(g.company_id)), 200


@bp.post("/zap")
@require_auth
@require_company_scope
def send_zap():
    """
    Кинуть заряд энергии коллеге, который сейчас в юните.
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Заряд отправлен}
    """
    try:
        data = _load(_zap_schema)
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    try:
        result = feed_service.send_zap(g.company_id, g.current_user.id,
                                       data["to_user_id"])
    except FeedServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(result), 200


# ───────────────────────────── питомец ─────────────────────────────

@bp.get("/pet")
@require_auth
@require_company_scope
def get_my_pet():
    """
    Мой Грувик (создаётся при первом обращении).
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Питомец}
    """
    return jsonify(pet_service.get_my_pet(g.current_user.id, g.company_id)), 200


@bp.post("/pet/feed")
@require_auth
@require_company_scope
def feed_pet():
    """
    Покормить Грувика (тратит грувы, даёт XP, двигает стрик).
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Питомец после кормления + реплика}
    """
    try:
        result = pet_service.feed_pet(g.current_user.id, g.company_id)
    except PetServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(result), 200


@bp.post("/pet/name")
@require_auth
@require_company_scope
def rename_pet():
    """
    Переименовать Грувика.
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Питомец}
    """
    try:
        data = _load(_rename_schema)
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    return jsonify(pet_service.rename_pet(g.current_user.id, g.company_id,
                                          data["name"])), 200


@bp.post("/pet/equip")
@require_auth
@require_company_scope
def equip_item():
    """
    Надеть/снять аксессуар (item: null — снять).
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Питомец}
    """
    try:
        data = _load(_equip_schema)
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    try:
        result = pet_service.equip_item(g.current_user.id, g.company_id,
                                        data.get("item"))
    except PetServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(result), 200


@bp.get("/shop")
@require_auth
def get_shop():
    """
    Прайс магазина аксессуаров.
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Словарь item → цена}
    """
    return jsonify(SHOP_PRICES), 200


@bp.post("/shop/buy")
@require_auth
@require_company_scope
def buy_item():
    """
    Купить аксессуар за грувы (сразу надевается).
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Питомец}
    """
    try:
        data = _load(_buy_schema)
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    try:
        result = pet_service.buy_item(g.current_user.id, g.company_id, data["item"])
    except PetServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(result), 200


# ─────────────────────────── зоопарк и рейд ────────────────────────

@bp.get("/zoo")
@require_auth
@require_company_scope
def get_zoo():
    """
    Зоопарк компании: все Грувики с поглаживаниями.
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Список питомцев}
    """
    return jsonify(pet_service.get_zoo(g.company_id, g.current_user.id)), 200


@bp.post("/zoo/<int:user_id>/stroke")
@require_auth
@require_company_scope
def stroke_pet(user_id: int):
    """
    Погладить Грувика коллеги (раз в день, грувы обоим).
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Счётчик поглаживаний за сегодня}
    """
    try:
        result = pet_service.stroke_pet(g.current_user.id, user_id, g.company_id)
    except PetServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(result), 200


@bp.get("/raid")
@require_auth
@require_company_scope
def get_raid():
    """
    Недельный рейд компании: босс, цель, прогресс.
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Состояние рейда}
    """
    return jsonify(pet_service.get_raid_state(g.company_id)), 200
