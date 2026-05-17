import json
import zipfile
from datetime import datetime, timezone
from io import BytesIO
from pathlib import Path

from app.extensions import db
from app.models import Role, User, Department, Task, Favorite, UnitType, Unit
from app.utils.logger import get_logger

logger = get_logger(__name__)


def _serialize_dt(dt) -> str | None:
    if dt is None:
        return None
    if isinstance(dt, datetime):
        return dt.isoformat()
    return str(dt)


def export_zip(upload_folder: str) -> BytesIO:
    data = {
        "roles": [
            {"id": r.id, "name": r.name, "level": r.level}
            for r in db.session.execute(db.select(Role)).scalars().all()
        ],
        "users": [
            {
                "id": u.id, "fio": u.fio, "login": u.login, "hash_password": u.hash_password,
                "post": u.post, "role_id": u.role_id, "avatar_path": u.avatar_path,
                "is_default_pass": u.is_default_pass, "is_hidden": u.is_hidden,
                "created_at": _serialize_dt(u.created_at),
            }
            for u in db.session.execute(db.select(User)).scalars().all()
        ],
        "departments": [
            {"id": d.id, "name": d.name}
            for d in db.session.execute(db.select(Department)).scalars().all()
        ],
        "tasks": [
            {
                "id": t.id, "name": t.name, "author_id": t.author_id,
                "link_yougile": t.link_yougile, "received_at": _serialize_dt(t.received_at),
                "department_id": t.department_id, "deadline": _serialize_dt(t.deadline),
                "is_archived": t.is_archived, "archived_at": _serialize_dt(t.archived_at),
                "created_at": _serialize_dt(t.created_at),
            }
            for t in db.session.execute(db.select(Task)).scalars().all()
        ],
        "favorites": [
            {"task_id": f.task_id, "user_id": f.user_id}
            for f in db.session.execute(db.select(Favorite)).scalars().all()
        ],
        "unit_types": [
            {"id": ut.id, "name": ut.name}
            for ut in db.session.execute(db.select(UnitType)).scalars().all()
        ],
        "units": [
            {
                "id": u.id, "name": u.name, "user_id": u.user_id, "unit_type_id": u.unit_type_id,
                "task_id": u.task_id, "is_edited": u.is_edited,
                "datetime_start": _serialize_dt(u.datetime_start),
                "datetime_end": _serialize_dt(u.datetime_end),
                "created_at": _serialize_dt(u.created_at),
            }
            for u in db.session.execute(db.select(Unit)).scalars().all()
        ],
    }

    buf = BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as zf:
        zf.writestr("data.json", json.dumps(data, ensure_ascii=False, indent=2))

        avatars_dir = Path(upload_folder) / "avatars"
        if avatars_dir.exists():
            for avatar_file in avatars_dir.iterdir():
                if avatar_file.is_file():
                    zf.write(avatar_file, f"avatars/{avatar_file.name}")

    buf.seek(0)
    logger.info("backup.export", extra={"extra": {"event": "backup.export"}})
    return buf


def import_zip(zip_bytes: bytes, upload_folder: str) -> None:
    with zipfile.ZipFile(BytesIO(zip_bytes)) as zf:
        data = json.loads(zf.read("data.json"))

        avatars_dir = Path(upload_folder) / "avatars"
        avatars_dir.mkdir(parents=True, exist_ok=True)
        for name in zf.namelist():
            if name.startswith("avatars/") and not name.endswith("/"):
                avatars_dir.joinpath(Path(name).name).write_bytes(zf.read(name))

    db.session.execute(db.text("TRUNCATE units, favorites, tasks, unit_types, departments, users, roles RESTART IDENTITY CASCADE"))

    for r in data.get("roles", []):
        db.session.execute(
            db.text("INSERT INTO roles (id, name, level) VALUES (:id, :name, :level)"),
            r
        )

    for u in data.get("users", []):
        db.session.execute(
            db.text("""
                INSERT INTO users (id, fio, login, hash_password, post, role_id, avatar_path,
                    is_default_pass, is_hidden, created_at)
                VALUES (:id, :fio, :login, :hash_password, :post, :role_id, :avatar_path,
                    :is_default_pass, :is_hidden, :created_at)
            """),
            u
        )

    for d in data.get("departments", []):
        db.session.execute(
            db.text("INSERT INTO departments (id, name) VALUES (:id, :name)"), d
        )

    for t in data.get("tasks", []):
        db.session.execute(
            db.text("""
                INSERT INTO tasks (id, name, author_id, link_yougile, received_at, department_id,
                    deadline, is_archived, archived_at, created_at)
                VALUES (:id, :name, :author_id, :link_yougile, :received_at, :department_id,
                    :deadline, :is_archived, :archived_at, :created_at)
            """),
            t
        )

    for f in data.get("favorites", []):
        db.session.execute(
            db.text("INSERT INTO favorites (task_id, user_id) VALUES (:task_id, :user_id)"), f
        )

    for ut in data.get("unit_types", []):
        db.session.execute(
            db.text("INSERT INTO unit_types (id, name) VALUES (:id, :name)"), ut
        )

    for u in data.get("units", []):
        db.session.execute(
            db.text("""
                INSERT INTO units (id, name, user_id, unit_type_id, task_id, is_edited,
                    datetime_start, datetime_end, created_at)
                VALUES (:id, :name, :user_id, :unit_type_id, :task_id, :is_edited,
                    :datetime_start, :datetime_end, :created_at)
            """),
            u
        )

    db.session.execute(db.text("SELECT setval('roles_id_seq', (SELECT MAX(id) FROM roles))"))
    db.session.execute(db.text("SELECT setval('users_id_seq', (SELECT MAX(id) FROM users))"))
    db.session.execute(db.text("SELECT setval('departments_id_seq', (SELECT MAX(id) FROM departments))"))
    db.session.execute(db.text("SELECT setval('tasks_id_seq', (SELECT MAX(id) FROM tasks))"))
    db.session.execute(db.text("SELECT setval('unit_types_id_seq', (SELECT MAX(id) FROM unit_types))"))
    db.session.execute(db.text("SELECT setval('units_id_seq', (SELECT MAX(id) FROM units))"))

    db.session.commit()
    logger.info("backup.import", extra={"extra": {"event": "backup.import"}})
