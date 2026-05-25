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
    other = user_repo.get_by_id(other_user_id)
    if other is None or other.is_hidden:
        raise MessengerServiceError("Собеседник не найден", "USER_NOT_FOUND", 404)
    conv = message_repo.get_or_create_conversation(current_user_id, other_user_id)
    db.session.commit()
    return conv


def _ensure_member(conv, user_id: int):
    if user_id not in (conv.user_a_id, conv.user_b_id):
        raise MessengerServiceError("Нет доступа к диалогу", "FORBIDDEN", 403)


def get_conversation_for_user(conversation_id: int, user_id: int):
    conv = message_repo.get_conversation(conversation_id)
    if conv is None:
        raise MessengerServiceError("Диалог не найден", "CONV_NOT_FOUND", 404)
    _ensure_member(conv, user_id)
    return conv


def send_message(conversation_id: int, sender_id: int,
                 text: Optional[str], attachment_ids: list[int]):
    conv = get_conversation_for_user(conversation_id, sender_id)
    text = (text or "").strip() or None
    attachment_ids = attachment_ids or []
    if not text and not attachment_ids:
        raise MessengerServiceError("Пустое сообщение", "EMPTY_MESSAGE", 400)

    # Проверим, что все вложения принадлежат отправителю и ещё не привязаны
    for att_id in attachment_ids:
        att = message_repo.get_attachment(att_id)
        if att is None or att.uploader_id != sender_id or att.message_id is not None:
            raise MessengerServiceError("Недопустимое вложение", "BAD_ATTACHMENT", 400)

    msg = message_repo.create_message(conversation_id, sender_id, text, attachment_ids)
    db.session.commit()
    logger.info("message.send", extra={"extra": {
        "event": "message.send", "conversation_id": conversation_id,
        "sender_id": sender_id, "message_id": msg.id,
    }})
    return conv, msg


def mark_conversation_read(conversation_id: int, user_id: int) -> int:
    conv = get_conversation_for_user(conversation_id, user_id)
    n = message_repo.mark_read(conv.id, user_id)
    db.session.commit()
    return n


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

    upload_folder = current_app.config["UPLOAD_FOLDER"]
    if not os.path.isabs(upload_folder):
        upload_folder = os.path.join(current_app.root_path, "..", upload_folder)
    abs_dir = os.path.abspath(os.path.join(upload_folder, rel_dir))
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
