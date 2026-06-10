"""Раздел «Мой Groove»: лента активности, реакции, комментарии,
питомцы-Грувики и командные рейды. Все таблицы — company-scoped.
"""
from datetime import datetime, timezone

from sqlalchemy.dialects.postgresql import JSONB

from app.extensions import db


class FeedEvent(db.Model):
    __tablename__ = "feed_events"

    id = db.Column(db.Integer, primary_key=True)
    company_id = db.Column(db.Integer, db.ForeignKey("companies.id", ondelete="CASCADE"),
                           nullable=False)
    # NULL — системное событие без автора-человека (AI-дайджест, рейд).
    user_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"),
                        nullable=True)
    # unit_started | unit_stopped | task_closed | streak | pet_evolved
    # | kudos | ai_digest | raid_started | raid_won
    kind = db.Column(db.String(32), nullable=False)
    payload = db.Column(JSONB, nullable=False, default=dict)
    created_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))

    user = db.relationship("User", foreign_keys=[user_id])
    reactions = db.relationship("FeedReaction", back_populates="event",
                                lazy="dynamic", cascade="all, delete-orphan")
    comments = db.relationship("FeedComment", back_populates="event",
                               lazy="dynamic", cascade="all, delete-orphan")

    __table_args__ = (
        # Курсорная пагинация ленты: WHERE company_id = ? AND id < ? ORDER BY id DESC
        db.Index("idx_feed_events_company_id", "company_id", "id"),
        db.Index("idx_feed_events_user", "user_id"),
    )


class FeedReaction(db.Model):
    __tablename__ = "feed_reactions"

    id = db.Column(db.Integer, primary_key=True)
    event_id = db.Column(db.Integer, db.ForeignKey("feed_events.id", ondelete="CASCADE"),
                         nullable=False)
    user_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"),
                        nullable=False)
    emoji = db.Column(db.String(16), nullable=False)
    created_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))

    event = db.relationship("FeedEvent", back_populates="reactions")

    __table_args__ = (
        db.UniqueConstraint("event_id", "user_id", "emoji", name="uq_feed_reaction"),
        db.Index("idx_feed_reactions_event", "event_id"),
    )


class FeedComment(db.Model):
    __tablename__ = "feed_comments"

    id = db.Column(db.Integer, primary_key=True)
    event_id = db.Column(db.Integer, db.ForeignKey("feed_events.id", ondelete="CASCADE"),
                         nullable=False)
    # NULL + is_bot=True — комментарий Грувика (AI-талисмана).
    author_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"),
                          nullable=True)
    is_bot = db.Column(db.Boolean, nullable=False, default=False)
    reply_to_id = db.Column(db.Integer, db.ForeignKey("feed_comments.id", ondelete="SET NULL"),
                            nullable=True)
    text = db.Column(db.Text, nullable=False)
    created_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))

    event = db.relationship("FeedEvent", back_populates="comments")
    author = db.relationship("User", foreign_keys=[author_id])
    reply_to = db.relationship("FeedComment", remote_side=[id], foreign_keys=[reply_to_id])

    __table_args__ = (
        db.Index("idx_feed_comments_event", "event_id", "created_at"),
    )


class Pet(db.Model):
    __tablename__ = "pets"

    user_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"),
                        primary_key=True)
    company_id = db.Column(db.Integer, db.ForeignKey("companies.id", ondelete="CASCADE"),
                           nullable=False)
    name = db.Column(db.String(50), nullable=False, default="Грувик")
    # egg (до вылупления) | owl | lark | sprinter | marathoner | fox
    species = db.Column(db.String(16), nullable=False, default="egg")
    stage = db.Column(db.Integer, nullable=False, default=0)
    xp = db.Column(db.Integer, nullable=False, default=0)
    beans = db.Column(db.Integer, nullable=False, default=0)
    # Надетый аксессуар + всё купленное/выигранное (список строк-ключей).
    hat = db.Column(db.String(32), nullable=True)
    accessories = db.Column(JSONB, nullable=False, default=list)
    feed_streak = db.Column(db.Integer, nullable=False, default=0)
    last_fed_date = db.Column(db.Date, nullable=True)
    # Болезнь: питомец заболевает, если хозяин долго не работал (нет
    # завершённых юнитов). Лечится работой и заботой (recovery-очки),
    # уровень и XP при этом НЕ теряются — болезнь лишь замораживает рост.
    sick_since = db.Column(db.DateTime(timezone=True), nullable=True)
    recovery = db.Column(db.Integer, nullable=False, default=0, server_default="0")
    # Характер: пересчитывается по паттерну работы (см. pet_service).
    personality = db.Column(db.String(24), nullable=True)
    # Виды, которые хозяин уже разблокировал (купил в магазине либо
    # естественно «развил» через эволюцию). Природный вид добавляется
    # автоматически, чтобы между ним и купленным можно было переключаться.
    unlocked_species = db.Column(JSONB, nullable=False, default=list,
                                 server_default="[]")
    # Ежедневный квест от Грувика: автообновление в полночь МСК.
    # kind: tasks_closed | units_finished | unit_minutes | feed_pet
    quest_date = db.Column(db.Date, nullable=True)
    quest_kind = db.Column(db.String(32), nullable=True)
    quest_target = db.Column(db.Integer, nullable=True)
    quest_progress = db.Column(db.Integer, nullable=False, default=0,
                               server_default="0")
    quest_claimed = db.Column(db.Boolean, nullable=False, default=False,
                              server_default="false")
    created_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))

    user = db.relationship("User", foreign_keys=[user_id])

    __table_args__ = (
        db.Index("idx_pets_company", "company_id"),
    )


class PetStroke(db.Model):
    __tablename__ = "pet_strokes"

    id = db.Column(db.Integer, primary_key=True)
    pet_user_id = db.Column(db.Integer, db.ForeignKey("pets.user_id", ondelete="CASCADE"),
                            nullable=False)
    user_id = db.Column(db.Integer, db.ForeignKey("users.id", ondelete="CASCADE"),
                        nullable=False)
    day = db.Column(db.Date, nullable=False)
    created_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))

    __table_args__ = (
        db.UniqueConstraint("pet_user_id", "user_id", "day", name="uq_pet_stroke_day"),
        db.Index("idx_pet_strokes_pet_day", "pet_user_id", "day"),
    )


class GrooveRaid(db.Model):
    __tablename__ = "groove_raids"

    id = db.Column(db.Integer, primary_key=True)
    company_id = db.Column(db.Integer, db.ForeignKey("companies.id", ondelete="CASCADE"),
                           nullable=False)
    week_start = db.Column(db.Date, nullable=False)
    boss = db.Column(db.String(64), nullable=False)
    target = db.Column(db.Integer, nullable=False)
    reward = db.Column(db.String(32), nullable=False, default="helmet")
    defeated_at = db.Column(db.DateTime(timezone=True), nullable=True)
    created_at = db.Column(db.DateTime(timezone=True), nullable=False,
                           default=lambda: datetime.now(timezone.utc))

    __table_args__ = (
        db.UniqueConstraint("company_id", "week_start", name="uq_raid_week"),
    )
