"""Бизнес-логика подключения пользователя к YouGile и работы с его ключом.

Поверх HTTP-клиента (`client.py`) и шифрования (`crypto.py`). Хранение —
`UserYougileAccount` (1:1 к user_id).

Поток для обычного пользователя:
  1. На фронте он видит, к какой YG-компании привязана GW-компания.
  2. Вводит свой YG-логин/пароль.
  3. `connect_finish_for_user(user, login, pwd)` сама проверяет, что у юзера
     в YG есть доступ к фиксированной yg_company_id; если нет — ошибка.
  4. Создаём ключ через YG, сохраняем зашифрованным.

Поток для админа (из company-визарда):
  1. `list_companies_with_credentials(login, pwd)` — выбираем yg_company_id.
  2. `connect_finish_for_admin(user, login, pwd, yg_company_id)` — то же
     самое, но company_id передаётся явно (т.к. в company.yg_company_id ещё
     может быть пусто — мы как раз сейчас его выбираем).
"""
from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timezone

from app.extensions import db
from app.integrations.yougile.client import (
    YougileAuthError, YougileClient, YougileError,
)
from app.integrations.yougile.crypto import (
    decrypt_key, encrypt_key, make_fingerprint,
)
from app.models.company import Company
from app.models.user import User
from app.models.user_yougile_account import UserYougileAccount
from app.utils.logger import get_logger

logger = get_logger(__name__)


class YougileAccountError(RuntimeError):
    """Пользовательская ошибка подключения (показываем как есть)."""
    def __init__(self, code: str, message: str):
        super().__init__(message)
        self.code = code
        self.message = message


@dataclass
class AccountStatus:
    connected: bool
    yg_login: str | None
    key_fingerprint: str | None
    last_validated_at: datetime | None
    yg_company_id: str | None
    company_enabled: bool


# ── статус ────────────────────────────────────────────────────────────────

def get_status(user: User) -> AccountStatus:
    company = user.company
    # `company_enabled` означает «интеграция реально работоспособна»:
    # включён флаг + директор выбрал компанию + есть доска + резолвлена
    # первая колонка. Если что-то из этого не задано, фронт показывает
    # старое простое поле «ссылка на задачу YouGile» и не пытается дёргать
    # импорт/экспорт.
    company_enabled = bool(
        company
        and (company.settings or {}).get("uses_yougile")
        and company.yg_company_id
        and company.yg_board_id
        and company.yg_first_column_id
    )
    acc: UserYougileAccount | None = (
        UserYougileAccount.query.filter_by(user_id=user.id).first()
    )
    if not acc:
        return AccountStatus(False, None, None, None, None, company_enabled)
    return AccountStatus(
        connected=True,
        yg_login=acc.yg_login,
        key_fingerprint=acc.key_fingerprint,
        last_validated_at=acc.last_validated_at,
        yg_company_id=acc.yg_company_id,
        company_enabled=company_enabled,
    )


# ── подключение ───────────────────────────────────────────────────────────

def list_companies_for_credentials(login: str, password: str) -> list[dict]:
    """Прозрачный прокси `/auth/companies`. Используется в админ-визарде.

    Все ошибки YG конвертим в YougileAccountError с понятным кодом.
    """
    client = YougileClient()
    try:
        items = client.list_companies(login, password)
    except YougileAuthError:
        raise YougileAccountError("BAD_CREDENTIALS", "Неверный логин или пароль")
    except YougileError as e:
        raise YougileAccountError("YOUGILE_ERROR", f"Ошибка YouGile: {e}")
    return [{"id": c.get("id"), "name": c.get("name")} for c in items if c.get("id")]


def _select_yg_company(login: str, password: str, target_id: str) -> dict | None:
    """Найти yg_company по id среди компаний пользователя.

    Возвращает None, если у пользователя нет такой компании — это явный
    сигнал, что админ ещё не пригласил его.
    """
    items = list_companies_for_credentials(login, password)
    for c in items:
        if c["id"] == target_id:
            return c
    return None


def connect_for_user(user: User, login: str, password: str,
                     explicit_yg_company_id: str | None = None) -> UserYougileAccount:
    """Универсальный коннект: используется и обычным юзером, и админом.

    Если `explicit_yg_company_id` задан — берём его (админ в визарде).
    Иначе — пытаемся взять `user.company.yg_company_id` (обычный юзер).
    """
    company = user.company
    if company is None:
        raise YougileAccountError(
            "NO_COMPANY",
            "Эта функция доступна только пользователям компании. "
            "Администратору системы нужно войти как директор конкретной компании.",
        )

    target_yg_id = explicit_yg_company_id or company.yg_company_id
    if not target_yg_id:
        raise YougileAccountError(
            "COMPANY_NOT_CONFIGURED",
            "В компании ещё не выбрана YouGile-компания — обратитесь к администратору",
        )

    yg_company = _select_yg_company(login, password, target_yg_id)
    if yg_company is None:
        raise YougileAccountError(
            "NO_ACCESS_TO_COMPANY",
            "У вашего аккаунта YouGile нет доступа к этой компании. "
            "Попросите администратора пригласить вас.",
        )

    # Шаг 2 — собственно ключ. /auth/keys на одних и тех же кредах возвращает
    # тот же ключ (по нашей проверке), но всё равно безопаснее перешифровать,
    # вдруг там логика изменится.
    anon = YougileClient()
    try:
        key = anon.create_key(login, password, target_yg_id)
    except YougileAuthError:
        raise YougileAccountError("BAD_CREDENTIALS", "Неверный логин или пароль")
    except YougileError as e:
        raise YougileAccountError("YOUGILE_ERROR", f"Ошибка YouGile: {e}")

    # Получаем yg_user_id — пригодится при «назначить себя» во время экспорта.
    auth = YougileClient(key=key)
    yg_user_id: str | None = None
    try:
        me = auth.me() or {}
        yg_user_id = me.get("id")
    except YougileError as e:
        # Не блокируем подключение, но логируем.
        logger.warning("yougile.me_failed_on_connect", extra={"user_id": user.id, "err": str(e)})

    # Если у юзера уже была старая запись и он переподключается с другим
    # логином — отзовём прошлый ключ в YG, чтобы не плодить ключи в его
    # аккаунте YG. Best-effort: если YG лежит, всё равно сохраним новую.
    existing: UserYougileAccount | None = UserYougileAccount.query.filter_by(user_id=user.id).first()
    if existing:
        old_key = decrypt_key(existing.key_ciphertext)
        if old_key and old_key != key:
            try:
                YougileClient().delete_key(old_key)
            except YougileError as e:
                logger.warning("yougile.old_key_revoke_failed",
                               extra={"user_id": user.id, "err": str(e)})

    if existing:
        existing.yg_company_id = target_yg_id
        existing.yg_user_id = yg_user_id
        existing.yg_login = login
        existing.key_ciphertext = encrypt_key(key)
        existing.key_fingerprint = make_fingerprint(key)
        existing.last_validated_at = datetime.now(timezone.utc)
        acc = existing
    else:
        acc = UserYougileAccount(
            user_id=user.id,
            company_id=company.id,
            yg_company_id=target_yg_id,
            yg_user_id=yg_user_id,
            yg_login=login,
            key_ciphertext=encrypt_key(key),
            key_fingerprint=make_fingerprint(key),
            last_validated_at=datetime.now(timezone.utc),
        )
        db.session.add(acc)
    db.session.commit()
    logger.info("yougile.connected", extra={"user_id": user.id, "yg_company_id": target_yg_id})
    return acc


def disconnect(user: User) -> None:
    """Отозвать ключ в YG и удалить локальную привязку."""
    acc = UserYougileAccount.query.filter_by(user_id=user.id).first()
    if not acc:
        return
    key = decrypt_key(acc.key_ciphertext)
    if key:
        try:
            YougileClient().delete_key(key)
        except YougileError as e:
            # Не блокируем удаление локальной записи — пользователь хочет
            # отвязаться, и должен это получить даже если YG недоступен.
            logger.warning("yougile.revoke_failed",
                           extra={"user_id": user.id, "err": str(e)})
    db.session.delete(acc)
    db.session.commit()
    logger.info("yougile.disconnected", extra={"user_id": user.id})


def rotate(user: User, password: str) -> UserYougileAccount:
    """Перевыпустить ключ. Принципиально требуем пароль повторно."""
    acc = UserYougileAccount.query.filter_by(user_id=user.id).first()
    if not acc:
        raise YougileAccountError("NOT_CONNECTED", "Аккаунт YouGile не подключён")
    # Используем уже сохранённый login + переданный заново password.
    return connect_for_user(user, acc.yg_login, password,
                            explicit_yg_company_id=acc.yg_company_id)


# ── клиент по запросу ─────────────────────────────────────────────────────

def build_client_for_user(user: User) -> YougileClient | None:
    """Готовый клиент с расшифрованным ключом или None.

    None означает «пользователь не подключён или ключ не расшифровался».
    Вызывающая сторона решает: отдать 412 «подключите YG» или просто молча
    пропустить (если, например, webhook применил изменение без действия юзера).
    """
    acc = UserYougileAccount.query.filter_by(user_id=user.id).first()
    if not acc:
        return None
    key = decrypt_key(acc.key_ciphertext)
    if not key:
        # ENC_KEY сменили — UI попросит переподключение.
        return None
    return YougileClient(key=key)
