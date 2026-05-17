import os
import uuid
import hashlib
import colorsys
from io import BytesIO
from pathlib import Path

import magic
from PIL import Image, ImageDraw


ALLOWED_MIME_TYPES = {"image/jpeg", "image/png"}
AVATAR_SUBDIR = "avatars"

# Сетка идентикона: 8 строк × 5 столбцов (3 уникальных + 2 зеркальных)
_ROWS = 8
_COLS = 5
_UNIQUE_COLS = 3
_CELL = 24          # px на ячейку → 120×192… нет, сделаем квадрат
_SIZE = 192         # итоговый размер после scale


def get_upload_dir(upload_folder: str) -> Path:
    path = Path(upload_folder) / AVATAR_SUBDIR
    path.mkdir(parents=True, exist_ok=True)
    return path


def _hsl_to_rgb(h: float, s: float, l: float):
    r, g, b = colorsys.hls_to_rgb(h, l, s)
    return (int(r * 255), int(g * 255), int(b * 255))


def generate_identicon(user_id: int, upload_folder: str) -> bytes:
    """Pixel-art identicon 8×8 с уникальной насыщенной цветовой схемой."""
    data = hashlib.sha256(str(user_id).encode()).digest()

    # Цвет переднего плана: насыщенный, по-настоящему уникальный оттенок
    hue = data[0] / 255.0
    fg = _hsl_to_rgb(hue, 0.70, 0.50)
    # Второй, более тёмный оттенок того же цвета для «тени»
    fg2 = _hsl_to_rgb(hue, 0.70, 0.35)
    bg = (248, 248, 252)

    # Рисуем на маленьком холсте 8×8, потом масштабируем — это и есть «pixel art»
    GRID = 8
    HALF = 4        # 4 уникальных столбца, 4 зеркальных
    SMALL = GRID    # ширина маленького холста = GRID px

    small = Image.new("RGB", (GRID, GRID), bg)
    pixels = small.load()

    for row in range(GRID):
        for col in range(HALF):
            idx = row * HALF + col
            byte = data[idx % len(data)]
            if byte > 100:
                color = fg2 if byte > 200 else fg
                pixels[col, row] = color
                pixels[GRID - 1 - col, row] = color

    # Масштабируем без сглаживания — получаем чёткий пиксель-арт
    img = small.resize((_SIZE, _SIZE), Image.NEAREST)

    buf = BytesIO()
    img.save(buf, format="PNG")
    return buf.getvalue()


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
