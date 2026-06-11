from .events import register_events

__all__ = ["register_events"]

# Подмодули sockets импортируются по точечной нотации (app.sockets.presence,
# app.sockets.call_events, app.sockets.call_bridge) — отдельные импорты тут
# не нужны. REST и бизнес-логика звонков живут в Go-микросервисе callsvc
# (back-go/calls); здесь только Socket.IO-шлюз ринг-фазы и Redis-мост событий.
