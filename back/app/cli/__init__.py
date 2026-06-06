from flask import Flask


def register_cli(app: Flask) -> None:
    from .ai import ai_group
    app.cli.add_command(ai_group)
