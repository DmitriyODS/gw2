from .events import register_events

__all__ = ["register_events"]

# Подмодули sockets импортируются по точечной нотации (app.sockets.presence,
# app.sockets.call_events, app.sockets.call_bridge, app.sockets.service_bridge)
# — отдельные импорты тут не нужны. REST и бизнес-логика звонков/мессенджера
# живут в Go-микросервисах (back-go/calls, back-go/messenger); здесь только
# Socket.IO-шлюз ринг-фазы и Redis-мосты событий.
