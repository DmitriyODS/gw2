from flask import Flask
from .roles import bp as roles_bp
from .tasks import bp as tasks_bp
from .units import bp as units_bp
from .departments import bp as departments_bp
from .unit_types import bp as unit_types_bp
from .stats import bp as stats_bp
from .backup import bp as backup_bp
from .changelog import bp as changelog_bp
from .messenger import bp as messenger_bp
from .companies import bp as companies_bp
from .stages import bp as stages_bp
from .ai_settings import bp as ai_settings_bp
from .ai_tv import bp as ai_tv_bp
from .yougile import bp as yougile_bp
from .groove import bp as groove_bp


def register_blueprints(app: Flask) -> None:
    app.register_blueprint(roles_bp)
    app.register_blueprint(tasks_bp)
    app.register_blueprint(units_bp)
    app.register_blueprint(departments_bp)
    app.register_blueprint(unit_types_bp)
    app.register_blueprint(stats_bp)
    app.register_blueprint(backup_bp)
    app.register_blueprint(changelog_bp)
    app.register_blueprint(messenger_bp)
    app.register_blueprint(companies_bp)
    app.register_blueprint(stages_bp)
    app.register_blueprint(ai_settings_bp)
    app.register_blueprint(ai_tv_bp)
    app.register_blueprint(yougile_bp)
    app.register_blueprint(groove_bp)
