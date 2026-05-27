from .events import register_events

__all__ = ["register_events"]

# Подмодули sockets уже импортируются по точечной нотации (app.sockets.presence,
# app.sockets.call_state, app.sockets.call_events) — отдельные импорты тут не
# нужны и приводят к циклу: __init__ → call_events → services.call_service → sockets.

