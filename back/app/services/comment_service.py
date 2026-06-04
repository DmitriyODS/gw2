from app.extensions import db
from app.repositories import comment_repo, task_repo, user_repo
from app.utils.permissions import get_user_level, MANAGER


class CommentServiceError(Exception):
    def __init__(self, message: str, code: str = "COMMENT_ERROR", http_status: int = 400):
        self.message = message
        self.code = code
        self.http_status = http_status
        super().__init__(message)


def _ensure_can_edit(comment, user_id: int):
    if comment.author_id == user_id:
        return
    user = user_repo.get_by_id(user_id)
    if user is None or get_user_level(user) < MANAGER:
        raise CommentServiceError("Нет прав на действие", "FORBIDDEN", 403)


def create_comment(task_id: int, author_id: int, text: str):
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise CommentServiceError("Задача не найдена", "TASK_NOT_FOUND", 404)
    text = (text or "").strip()
    if not text:
        raise CommentServiceError("Пустой текст", "EMPTY", 422)
    comment = comment_repo.create(task_id=task_id, author_id=author_id, text=text)
    db.session.commit()
    return comment


def list_comments(task_id: int):
    task = task_repo.get_by_id(task_id)
    if task is None:
        raise CommentServiceError("Задача не найдена", "TASK_NOT_FOUND", 404)
    return comment_repo.list_by_task(task_id)


def update_comment(comment_id: int, user_id: int, text: str):
    comment = comment_repo.get_by_id(comment_id)
    if comment is None or comment.deleted_at is not None:
        raise CommentServiceError("Комментарий не найден", "NOT_FOUND", 404)
    _ensure_can_edit(comment, user_id)
    text = (text or "").strip()
    if not text:
        raise CommentServiceError("Пустой текст", "EMPTY", 422)
    comment_repo.update_text(comment, text)
    db.session.commit()
    return comment


def delete_comment(comment_id: int, user_id: int):
    comment = comment_repo.get_by_id(comment_id)
    if comment is None or comment.deleted_at is not None:
        raise CommentServiceError("Комментарий не найден", "NOT_FOUND", 404)
    _ensure_can_edit(comment, user_id)
    comment_repo.soft_delete(comment)
    db.session.commit()
    return comment
