import os
import uuid
from datetime import datetime, timezone
from typing import Optional

from flask import current_app

from app.extensions import db
from app.repositories import message_repo, user_repo
from app.utils.logger import get_logger

logger = get_logger(__name__)


class MessengerServiceError(Exception):
    def __init__(self, message: str, code: str = "MESSENGER_ERROR", http_status: int = 400):
        self.message = message
        self.code = code
        self.http_status = http_status
        super().__init__(message)


# Разрешённые MIME-категории и максимальный размер берём из конфига приложения.
_ALLOWED_MIME_PREFIXES = ("image/", "audio/", "video/", "application/", "text/")


def open_conversation(current_user_id: int, other_user_id: int):
    if current_user_id == other_user_id:
        raise MessengerServiceError("Нельзя написать самому себе", "SELF_CONVERSATION", 400)
    me = user_repo.get_by_id(current_user_id)
    other = user_repo.get_by_id(other_user_id)
    if other is None or other.is_hidden:
        raise MessengerServiceError("Собеседник не найден", "USER_NOT_FOUND", 404)
    # Multi-tenancy: вне компании писать нельзя. Администратор системы
    # (company_id IS NULL) может писать любому сотруднику любой компании.
    # Сотрудник может писать только тем, кто в его же компании или
    # Администратору системы.
    if me is not None and me.company_id is not None and other.company_id is not None:
        if me.company_id != other.company_id:
            raise MessengerServiceError(
                "Нельзя писать сотруднику другой компании", "CROSS_COMPANY", 403,
            )
    conv = message_repo.get_or_create_conversation(current_user_id, other_user_id)
    db.session.commit()
    return conv


def open_dev_chat(current_user_id: int):
    """Открыть/создать спец-чат компании с разработчиками. У каждого сотрудника
    свой спец-чат через его company. Администратор системы вызывать не должен
    (он использует list_dev_chats / открывает чат конкретной компании)."""
    me = user_repo.get_by_id(current_user_id)
    if me is None:
        raise MessengerServiceError("Пользователь не найден", "USER_NOT_FOUND", 404)
    if me.company_id is None:
        raise MessengerServiceError(
            "У Администратора системы нет своего спец-чата",
            "ADMIN_HAS_NO_DEVCHAT", 400,
        )
    conv = message_repo.get_or_create_dev_chat(me.company_id)
    db.session.commit()
    return conv


def open_dev_chat_for_company(current_user_id: int, company_id: int):
    """Открыть/создать спец-чат для компании от имени Администратора системы."""
    me = user_repo.get_by_id(current_user_id)
    if me is None or me.company_id is not None:
        raise MessengerServiceError("Только Администратор системы", "FORBIDDEN", 403)
    conv = message_repo.get_or_create_dev_chat(company_id)
    db.session.commit()
    return conv


def _ensure_member(conv, user_id: int):
    """Проверяет доступ к диалогу. Для p2p — только участники. Для dev-чата —
    сотрудники компании chat'а + Администраторы системы."""
    if conv.is_dev_chat:
        user = user_repo.get_by_id(user_id)
        if user is None:
            raise MessengerServiceError("Нет доступа к диалогу", "FORBIDDEN", 403)
        if user.company_id is None:
            return  # Администратор системы
        if user.company_id != conv.company_id:
            raise MessengerServiceError("Нет доступа к диалогу", "FORBIDDEN", 403)
        return
    if user_id not in (conv.user_a_id, conv.user_b_id):
        raise MessengerServiceError("Нет доступа к диалогу", "FORBIDDEN", 403)


def get_conversation_for_user(conversation_id: int, user_id: int):
    conv = message_repo.get_conversation(conversation_id)
    if conv is None:
        raise MessengerServiceError("Диалог не найден", "CONV_NOT_FOUND", 404)
    _ensure_member(conv, user_id)
    return conv


def send_message(conversation_id: int, sender_id: int,
                 text: Optional[str], attachment_ids: list[int],
                 reply_to_id: Optional[int] = None,
                 task_id: Optional[int] = None):
    conv = get_conversation_for_user(conversation_id, sender_id)
    text = (text or "").strip() or None
    attachment_ids = attachment_ids or []
    if not text and not attachment_ids and task_id is None:
        raise MessengerServiceError("Пустое сообщение", "EMPTY_MESSAGE", 400)

    # Проверим, что все вложения принадлежат отправителю и ещё не привязаны
    for att_id in attachment_ids:
        att = message_repo.get_attachment(att_id)
        if att is None or att.uploader_id != sender_id or att.message_id is not None:
            raise MessengerServiceError("Недопустимое вложение", "BAD_ATTACHMENT", 400)

    # Ответ должен указывать на сообщение этого же диалога.
    if reply_to_id is not None:
        target = message_repo.get_message(reply_to_id)
        if target is None or target.conversation_id != conv.id:
            raise MessengerServiceError("Недопустимый ответ", "BAD_REPLY", 400)

    # Прикреплённая задача: должна быть из той же компании, что и диалог.
    kind = "text"
    if task_id is not None:
        from app.repositories import task_repo
        task = task_repo.get_by_id(task_id)
        if task is None:
            raise MessengerServiceError("Задача не найдена", "TASK_NOT_FOUND", 404)
        if conv.company_id and task.company_id != conv.company_id:
            raise MessengerServiceError(
                "Задача из другой компании", "TASK_WRONG_COMPANY", 400,
            )
        kind = "task"

    # В dev-чате ответ Администратора системы получает специальный kind —
    # фронт рисует «Разработчики» badge. Для kind='task' это правило не
    # применяется (плашка задачи имеет приоритет).
    if conv.is_dev_chat and kind == "text":
        sender = user_repo.get_by_id(sender_id)
        if sender is not None and sender.company_id is None:
            kind = "system_dev_reply"

    msg = message_repo.create_message(conversation_id, sender_id, text, attachment_ids,
                                      reply_to_id=reply_to_id, kind=kind, task_id=task_id)
    db.session.commit()
    logger.info("message.send", extra={"extra": {
        "event": "message.send", "conversation_id": conversation_id,
        "sender_id": sender_id, "message_id": msg.id,
    }})
    return conv, msg


def forward_message(source_message_id: int, sender_id: int,
                    conversation_ids: list[int], user_ids: list[int]):
    """Пересылает сообщение в один или несколько диалогов. Текст и файлы
    копируются (файлы — физически, чтобы удаление одной копии не задевало
    другую). user_ids — адресаты, для которых диалог создаётся при отсутствии.
    Возвращает список (conversation, message) по каждому адресату."""
    src = message_repo.get_message(source_message_id)
    if src is None:
        raise MessengerServiceError("Сообщение не найдено", "MSG_NOT_FOUND", 404)
    src_conv = message_repo.get_conversation(src.conversation_id)
    if src_conv is None or sender_id not in (src_conv.user_a_id, src_conv.user_b_id):
        raise MessengerServiceError("Нет доступа к сообщению", "FORBIDDEN", 403)

    # Соберём целевые диалоги: явные id + диалоги с указанными пользователями.
    target_convs: list = []
    seen_ids: set[int] = set()
    for cid in conversation_ids or []:
        conv = get_conversation_for_user(cid, sender_id)
        # Пересылать в dev-чат смысла нет (это поток в техподдержку).
        if conv.is_dev_chat:
            continue
        if conv.id not in seen_ids:
            target_convs.append(conv)
            seen_ids.add(conv.id)
    for uid in user_ids or []:
        conv = open_conversation(sender_id, uid)  # внутри проверка company scope
        if conv.id not in seen_ids:
            target_convs.append(conv)
            seen_ids.add(conv.id)

    if not target_convs:
        raise MessengerServiceError("Не выбран получатель", "NO_TARGET", 400)

    # Автор оригинала — кого показать в метке «Переслано от …».
    origin_user_id = src.forwarded_from_user_id or src.sender_id

    results = []
    for conv in target_convs:
        new_att_ids = []
        for att in src.attachments:
            copied = _copy_attachment(att, sender_id)
            new_att_ids.append(copied.id)
        msg = message_repo.create_message(
            conv.id, sender_id, src.text, new_att_ids,
            forwarded_from_user_id=origin_user_id,
        )
        results.append((conv, msg))

    db.session.commit()
    logger.info("message.forward", extra={"extra": {
        "event": "message.forward", "source_message_id": source_message_id,
        "sender_id": sender_id, "targets": [c.id for c in target_convs],
    }})
    return results


def _copy_attachment(att, uploader_id: int):
    """Физически копирует файл вложения и регистрирует новую запись (message_id
    проставится при create_message). Возвращает новый attachment."""
    src_abs = _abs_upload_path(att.file_path)
    ext = os.path.splitext(att.file_path)[1].lower()[:16]
    safe_name = f"{uuid.uuid4().hex}{ext}"
    rel_dir = os.path.join("messages", datetime.now(timezone.utc).strftime("%Y/%m"))
    abs_dir = _abs_upload_path(rel_dir)
    os.makedirs(abs_dir, exist_ok=True)
    rel_path = os.path.join(rel_dir, safe_name).replace(os.sep, "/")
    dst_abs = os.path.join(abs_dir, safe_name)
    try:
        with open(src_abs, "rb") as fsrc, open(dst_abs, "wb") as fdst:
            fdst.write(fsrc.read())
    except OSError as e:
        raise MessengerServiceError("Не удалось скопировать вложение", "COPY_FAILED", 500) from e
    return message_repo.create_attachment(
        uploader_id=uploader_id,
        file_path=rel_path,
        file_name=att.file_name,
        mime_type=att.mime_type,
        size_bytes=att.size_bytes,
    )


def mark_conversation_read(conversation_id: int, user_id: int) -> int:
    conv = get_conversation_for_user(conversation_id, user_id)
    n = message_repo.mark_read(conv.id, user_id)
    db.session.commit()
    return n


def _abs_upload_path(rel_path: str) -> str:
    upload_folder = current_app.config["UPLOAD_FOLDER"]
    if not os.path.isabs(upload_folder):
        upload_folder = os.path.join(current_app.root_path, "..", upload_folder)
    return os.path.abspath(os.path.join(upload_folder, rel_path))


def _delete_attachment_files(paths: list[str]) -> None:
    for p in paths:
        if not p:
            continue
        try:
            abs_path = _abs_upload_path(p)
            if os.path.isfile(abs_path):
                os.remove(abs_path)
        except OSError as e:
            logger.warning("attachment.unlink_failed", extra={"extra": {"path": p, "error": str(e)}})


def delete_message(message_id: int, user_id: int, scope: str) -> tuple[int, bool]:
    """Удаляет сообщение. scope: 'me' (скрыть на своей стороне) или 'all'
    (физически удалить — только для своих сообщений).
    Возвращает (conversation_id, deleted_for_all_now)."""
    msg = message_repo.get_message(message_id)
    if msg is None:
        raise MessengerServiceError("Сообщение не найдено", "MSG_NOT_FOUND", 404)
    conv = message_repo.get_conversation(msg.conversation_id)
    if conv is None or user_id not in (conv.user_a_id, conv.user_b_id):
        raise MessengerServiceError("Нет доступа к сообщению", "FORBIDDEN", 403)

    physically_removed = False

    if scope == 'all':
        if msg.sender_id != user_id:
            raise MessengerServiceError(
                "Удалить «для всех» можно только своё сообщение", "FORBIDDEN", 403,
            )
        paths = [a.file_path for a in msg.attachments]
        message_repo.delete_message(msg)
        physically_removed = True
        message_repo.recompute_last_message_at(conv.id)
        db.session.commit()
        _delete_attachment_files(paths)
    elif scope == 'me':
        side = conv.side(user_id)
        both = message_repo.hide_message_for(msg, side)
        if both:
            paths = [a.file_path for a in msg.attachments]
            message_repo.delete_message(msg)
            physically_removed = True
            message_repo.recompute_last_message_at(conv.id)
            db.session.commit()
            _delete_attachment_files(paths)
        else:
            db.session.commit()
    else:
        raise MessengerServiceError("Неверный scope", "BAD_SCOPE", 400)

    logger.info("message.delete", extra={"extra": {
        "event": "message.delete", "message_id": message_id,
        "user_id": user_id, "scope": scope,
    }})
    return conv.id, (scope == 'all') or physically_removed


def delete_conversation(conversation_id: int, user_id: int, scope: str) -> bool:
    """Удаляет диалог. scope: 'me' (скрыть у себя — собеседник продолжит
    видеть переписку до своего удаления) или 'all' (физически удалить
    у обоих). Возвращает True, если диалог физически удалён."""
    conv = get_conversation_for_user(conversation_id, user_id)

    if scope == 'all':
        paths = message_repo.list_attachment_paths_of_conversation(conv.id)
        message_repo.delete_conversation(conv)
        db.session.commit()
        _delete_attachment_files(paths)
        physically_removed = True
    elif scope == 'me':
        side = conv.side(user_id)
        both = message_repo.hide_conversation_for(conv, side)
        if both:
            paths = message_repo.list_attachment_paths_of_conversation(conv.id)
            message_repo.delete_conversation(conv)
            db.session.commit()
            _delete_attachment_files(paths)
            physically_removed = True
        else:
            db.session.commit()
            physically_removed = False
    else:
        raise MessengerServiceError("Неверный scope", "BAD_SCOPE", 400)

    logger.info("conversation.delete", extra={"extra": {
        "event": "conversation.delete", "conversation_id": conversation_id,
        "user_id": user_id, "scope": scope, "physical": physically_removed,
    }})
    return physically_removed


def toggle_pin(conversation_id: int, user_id: int) -> bool:
    """Переключает закрепление диалога у пользователя. Возвращает новое
    состояние (True = закреплён)."""
    conv = get_conversation_for_user(conversation_id, user_id)
    side = conv.side(user_id)
    current = conv.pinned_at_for(user_id)
    message_repo.set_pin(conv, side, pinned=current is None)
    db.session.commit()
    return current is None


def toggle_message_pin(message_id: int, user_id: int):
    """Закрепить/открепить сообщение в диалоге. Закрепление общее: видят оба
    участника. Возвращает (conversation, message, pinned)."""
    msg = message_repo.get_message(message_id)
    if msg is None:
        raise MessengerServiceError("Сообщение не найдено", "MSG_NOT_FOUND", 404)
    conv = message_repo.get_conversation(msg.conversation_id)
    if conv is None or user_id not in (conv.user_a_id, conv.user_b_id):
        raise MessengerServiceError("Нет доступа к сообщению", "FORBIDDEN", 403)
    # Системные плашки звонка закреплять незачем.
    if msg.kind != "text":
        raise MessengerServiceError("Это сообщение нельзя закрепить", "BAD_PIN", 400)

    pinned = msg.pinned_at is None
    message_repo.set_message_pin(msg, pinned=pinned, by_id=user_id if pinned else None)
    db.session.commit()
    return conv, msg, pinned


def list_pinned_messages(conversation_id: int, user_id: int):
    """Закреплённые сообщения диалога (для баннера сверху)."""
    get_conversation_for_user(conversation_id, user_id)
    return message_repo.list_pinned_messages(conversation_id, user_id)


def upload_attachment(uploader_id: int, file_storage) -> dict:
    """Сохраняет файл на диск, регистрирует attachment, возвращает запись.
    file_storage — werkzeug FileStorage из request.files."""
    max_size = current_app.config.get("MESSENGER_ATTACHMENT_MAX", 25 * 1024 * 1024)

    data = file_storage.read()
    if not data:
        raise MessengerServiceError("Пустой файл", "EMPTY_FILE", 400)
    if len(data) > max_size:
        raise MessengerServiceError(
            f"Файл превышает {max_size // (1024 * 1024)} МБ", "FILE_TOO_LARGE", 400,
        )

    mime = (file_storage.mimetype or "application/octet-stream").lower()
    if not any(mime.startswith(p) for p in _ALLOWED_MIME_PREFIXES):
        raise MessengerServiceError("Неподдерживаемый тип файла", "BAD_MIME", 400)

    original = file_storage.filename or "file"
    ext = os.path.splitext(original)[1].lower()[:16]
    safe_name = f"{uuid.uuid4().hex}{ext}"
    rel_dir = os.path.join("messages", datetime.now(timezone.utc).strftime("%Y/%m"))
    abs_dir = _abs_upload_path(rel_dir)
    os.makedirs(abs_dir, exist_ok=True)
    abs_path = os.path.join(abs_dir, safe_name)
    with open(abs_path, "wb") as f:
        f.write(data)

    rel_path = os.path.join(rel_dir, safe_name).replace(os.sep, "/")
    att = message_repo.create_attachment(
        uploader_id=uploader_id,
        file_path=rel_path,
        file_name=original[:255],
        mime_type=mime[:120],
        size_bytes=len(data),
    )
    db.session.commit()
    return att
