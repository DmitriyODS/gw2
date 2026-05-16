import logging
import json
import os


class JSONFormatter(logging.Formatter):
    def format(self, record: logging.LogRecord) -> str:
        payload = {
            "ts": self.formatTime(record, "%Y-%m-%dT%H:%M:%SZ"),
            "level": record.levelname,
            "event": record.getMessage(),
        }
        if hasattr(record, "extra"):
            payload.update(record.extra)
        return json.dumps(payload, ensure_ascii=False)


def get_logger(name: str) -> logging.Logger:
    logger = logging.getLogger(name)
    if not logger.handlers:
        handler = logging.StreamHandler()
        handler.setFormatter(JSONFormatter())
        logger.addHandler(handler)
    level = logging.DEBUG if os.getenv("FLASK_DEBUG") else logging.INFO
    logger.setLevel(level)
    return logger
