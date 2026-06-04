from flask import Flask
from .auth import bp as auth_bp
from .users import bp as users_bp
from .roles import bp as roles_bp
from .tasks import bp as tasks_bp
from .units import bp as units_bp
from .departments import bp as departments_bp
from .unit_types import bp as unit_types_bp
from .stats import bp as stats_bp
from .backup import bp as backup_bp
from .changelog import bp as changelog_bp
from .messenger import bp as messenger_bp
from .calls import bp as calls_bp
from .companies import bp as companies_bp


def register_blueprints(app: Flask) -> None:
    app.register_blueprint(auth_bp)
    app.register_blueprint(users_bp)
    app.register_blueprint(roles_bp)
    app.register_blueprint(tasks_bp)
    app.register_blueprint(units_bp)
    app.register_blueprint(departments_bp)
    app.register_blueprint(unit_types_bp)
    app.register_blueprint(stats_bp)
    app.register_blueprint(backup_bp)
    app.register_blueprint(changelog_bp)
    app.register_blueprint(messenger_bp)
    app.register_blueprint(calls_bp)
    app.register_blueprint(companies_bp)
