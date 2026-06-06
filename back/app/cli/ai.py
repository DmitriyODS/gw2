"""CLI: AI-административные команды.

Запуск: `flask ai <subcommand> [--options]`.
"""
import click
from flask.cli import AppGroup

from app.services.task_embedding_service import run_backfill


ai_group = AppGroup("ai", help="Утилиты для AI-функциональности")


@ai_group.command("backfill-embeddings",
                  help="Сгенерировать недостающие эмбеддинги задач.")
@click.option("--company-id", type=int, default=None,
              help="Ограничить одной компанией (по умолчанию — все с включённым AI).")
def backfill_embeddings(company_id: int | None):
    click.echo(f"Backfill: company_id={company_id or 'all'}")
    stats = run_backfill(company_id)
    click.echo(f"Всего без индекса: {stats['total']}")
    click.echo(f"Проиндексировано: {stats['indexed']}")
    if stats["total"] and stats["indexed"] < stats["total"]:
        click.echo(
            click.style(
                "Часть задач не проиндексировалась — проверьте логи "
                "(возможно, упал AI-клиент). Повторный запуск догонит их.",
                fg="yellow",
            )
        )
