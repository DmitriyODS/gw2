"""Автоответ техподдержки в dev-чате.

Первое за сутки сообщение владельца чата получает автоответ-бот («сообщение
направлено разработчикам»); повторные в течение суток — нет. Тест создаёт
собственного пользователя со свежим dev-чатом, чтобы не зависеть от истории
переписки в общей dev-БД, и подчищает за собой.
"""
import uuid

import pytest


@pytest.fixture
def fresh_user(app):
    """Временный сотрудник существующей компании (свежий dev-чат гарантирован)."""
    from app.extensions import db
    from app.models import Company, Conversation, Message, User

    with app.app_context():
        company_id = db.session.execute(
            db.select(Company.id).order_by(Company.id).limit(1)
        ).scalar_one_or_none()
        if company_id is None:
            pytest.skip("В БД нет ни одной компании")
        user = User(
            fio="Тест Автоответа",
            login=f"test-auto-reply-{uuid.uuid4().hex[:10]}",
            hash_password="x",
            role_id=1,
            company_id=company_id,
            is_default_pass=False,
        )
        db.session.add(user)
        db.session.commit()
        uid = user.id

    yield uid

    with app.app_context():
        conv_ids = db.session.execute(
            db.select(Conversation.id).where(Conversation.user_a_id == uid)
        ).scalars().all()
        if conv_ids:
            db.session.execute(db.delete(Message).where(Message.conversation_id.in_(conv_ids)))
            db.session.execute(db.delete(Conversation).where(Conversation.id.in_(conv_ids)))
        db.session.execute(db.delete(User).where(User.id == uid))
        db.session.commit()


def test_support_auto_reply_once_per_day(app, fresh_user):
    from app.repositories import message_repo
    from app.services import messenger_service

    with app.app_context():
        conv = messenger_service.open_dev_chat(fresh_user)

        # Первое обращение — бот отвечает.
        conv, msg1 = messenger_service.send_message(conv.id, fresh_user, "Помогите!", [])
        auto = messenger_service.maybe_support_auto_reply(conv, msg1)
        assert auto is not None
        assert auto.is_bot and auto.sender_id is None
        assert auto.kind == "system_dev_reply"
        assert auto.text == messenger_service.SUPPORT_AUTO_REPLY_TEXT

        # Повторное сообщение в течение суток — без автоответа.
        conv, msg2 = messenger_service.send_message(conv.id, fresh_user, "Ещё деталь", [])
        assert messenger_service.maybe_support_auto_reply(conv, msg2) is None

        # Бот-сообщение учитывается в непрочитанных владельца (sender NULL).
        items = message_repo.list_user_conversations(fresh_user)
        dev_item = next(i for i in items if i["conversation"].id == conv.id)
        assert dev_item["unread_count"] >= 1
