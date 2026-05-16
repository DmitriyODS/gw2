import os
import uuid
import hashlib
from pathlib import Path

import pydenticon
import magic


ALLOWED_MIME_TYPES = {"image/jpeg", "image/png"}
AVATAR_SUBDIR = "avatars"


def get_upload_dir(upload_folder: str) -> Path:
    path = Path(upload_folder) / AVATAR_SUBDIR
    path.mkdir(parents=True, exist_ok=True)
    return path


def generate_identicon(user_id: int, upload_folder: str) -> bytes:
    """Генерировать identicon PNG по user_id (GitHub-style 5×5)."""
    generator = pydenticon.Generator(
        5, 5,
        foreground=["rgb(45,79,255)", "rgb(254,180,44)", "rgb(226,121,234)",
                    "rgb(30,179,253)", "rgb(232,77,65)"],
        background="rgb(240,240,240)",
    )
    hash_input = str(user_id).encode("utf-8")
    identicon_bytes = generator.generate(
        hashlib.md5(hash_input).hexdigest(), 200, 200,
        output_format="png"
    )
    return identicon_bytes


def validate_image(file_bytes: bytes) -> str:
    """Проверить MIME-тип файла. Возвращает mime или бросает ValueError."""
    mime = magic.from_buffer(file_bytes, mime=True)
    if mime not in ALLOWED_MIME_TYPES:
        raise ValueError(f"Недопустимый тип файла: {mime}. Разрешены: JPEG, PNG")
    return mime


def save_avatar(file_bytes: bytes, upload_folder: str) -> str:
    """Сохранить аватарку. Возвращает относительный путь avatars/{uuid}.ext."""
    mime = validate_image(file_bytes)
    ext = "jpg" if mime == "image/jpeg" else "png"
    filename = f"{uuid.uuid4()}.{ext}"
    upload_dir = get_upload_dir(upload_folder)
    filepath = upload_dir / filename
    filepath.write_bytes(file_bytes)
    return f"{AVATAR_SUBDIR}/{filename}"


def delete_avatar(avatar_path: str, upload_folder: str) -> None:
    """Удалить файл аватарки с диска. Молча игнорирует если файл не найден."""
    if not avatar_path:
        return
    full_path = Path(upload_folder) / avatar_path
    if full_path.exists():
        full_path.unlink()
