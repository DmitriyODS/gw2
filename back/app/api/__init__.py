from flask import Flask
from .messenger import bp as messenger_bp
from .yougile import bp as yougile_bp


def register_blueprints(app: Flask) -> None:
    app.register_blueprint(messenger_bp)
    app.register_blueprint(yougile_bp)
