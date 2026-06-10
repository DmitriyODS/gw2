"""Раздел «Мой Groove»: лента, реакции, комментарии, кудосы, заряды,
питомцы-Грувики, зоопарк, магазин и недельный рейд."""
from flask import Blueprint, g, jsonify, request
from marshmallow import ValidationError

from app.schemas.groove import (FeedReactionToggleSchema, FeedCommentCreateSchema,
                                KudosSchema, ZapSchema, PetRenameSchema,
                                ShopBuySchema, PetEquipSchema, PetSpeciesSchema,
                                FEED_REACTIONS)
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
_species_schema = PetSpeciesSchema()


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
    «Сейчас в эфире» — активные юниты компании с зарядами
    + личный остаток зарядов на сегодня.
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Активные юниты (items) + zaps_left/zaps_max}
    """
    return jsonify(feed_service.get_live(g.company_id, g.current_user.id)), 200


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
    Магазин аксессуаров: прайс + сезонный товар.
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Прайс, сезонный товар и название сезона}
    """
    return jsonify(pet_service.get_shop_state()), 200


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


@bp.post("/shop/buy-species")
@require_auth
@require_company_scope
def buy_species():
    """
    Купить новый облик Грувика (виды-зверюшки) — сразу надевается.
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Питомец}
    """
    try:
        data = _load(_species_schema)
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    try:
        result = pet_service.buy_species(g.current_user.id, g.company_id,
                                          data["species"])
    except PetServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(result), 200


@bp.post("/pet/quest/claim")
@require_auth
@require_company_scope
def claim_quest():
    """
    Забрать награду за выполненный квест дня (+бонус-грувы).
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Питомец после получения награды}
    """
    try:
        result = pet_service.claim_quest(g.current_user.id, g.company_id)
    except PetServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify(result), 200


@bp.post("/pet/species")
@require_auth
@require_company_scope
def switch_species():
    """
    Сменить облик Грувика на уже разблокированный (без оплаты).
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Питомец}
    """
    try:
        data = _load(_species_schema)
    except ValidationError as e:
        return jsonify({"error": "VALIDATION_ERROR", "message": e.messages}), 400
    try:
        result = pet_service.switch_species(g.current_user.id, g.company_id,
                                             data["species"])
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


# ──────────────────── wrapped и ТВ-витрина ─────────────────────────

@bp.get("/wrapped")
@require_auth
@require_company_scope
def get_wrapped():
    """
    «Моя неделя» — личный итог последних 7 дней (карточки-истории).
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Статистика недели + AI-фраза}
    """
    return jsonify(feed_service.get_wrapped(g.company_id, g.current_user.id)), 200


@bp.post("/wrapped/share")
@require_auth
@require_company_scope
def share_wrapped():
    """
    Опубликовать итог недели в ленту (раз в день).
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      201: {description: Итог опубликован}
    """
    try:
        feed_service.share_wrapped(g.company_id, g.current_user.id)
    except FeedServiceError as e:
        return jsonify({"error": e.code, "message": e.message}), e.http_status
    return jsonify({"message": "Итог недели опубликован"}), 201


@bp.get("/tv")
@require_auth
@require_company_scope
def groove_tv():
    """
    Данные для ТВ-слайда «Мой Groove»: топ Грувиков, рейд, итоги.
    ---
    tags: [groove]
    security: [BearerAuth: []]
    responses:
      200: {description: Топ питомцев + рейд + тоталы}
    """
    from app.repositories import pet_repo
    from app.services.pet_service import _today_msk, dump_pet

    pets = pet_repo.list_company_pets(g.company_id)
    strokes = pet_repo.strokes_today([p.user_id for p in pets], _today_msk())
    top = []
    for p in pets[:8]:
        data = dump_pet(p)
        data["strokes_today"] = strokes.get(p.user_id, 0)
        top.append(data)
    return jsonify({
        "pets": top,
        "raid": pet_service.get_raid_state(g.company_id),
        "totals": {
            "pets": len(pets),
            "sick": sum(1 for p in pets if p.sick_since is not None),
            "beans": sum(p.beans for p in pets),
            "strokes_today": sum(strokes.values()),
        },
    }), 200
