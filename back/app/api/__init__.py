from flask import Flask
from .messenger import bp as messenger_bp


def register_blueprints(app: Flask) -> None:
    app.register_blueprint(messenger_bp)
