// migrate — миграции схемы общей PostgreSQL Groove Work (goose, embed).
//
// Запускается отдельным run-once контейнером ДО старта сервисов
// (docker-compose: depends_on condition service_completed_successfully) —
// наследник `flask db upgrade` из entrypoint'а прежнего app-контейнера.
//
// Переход с Alembic: ревизия 00001 — baseline-снимок всей схемы на момент
// фазы 5. БД, жившая на Alembic (есть таблица alembic_version — в том числе
// восстановленный дамп прод-БД), получает baseline ПОМЕЧЕННЫМ применённым
// без выполнения SQL; свежая БД накатывает его целиком. Дальше обе идут
// обычным goose up.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	migrate "github.com/DmitriyODS/gw2/back-go/migrate"
)

const baselineVersion = 1

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	dbURL := env("DATABASE_URL", "postgresql://grovework:grovework_local@localhost:5432/grovework")

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Error("migrate.open_failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// БД может ещё подниматься (compose стартует после healthy db, но
	// перестраховка дешёвая).
	ctx := context.Background()
	if err := waitForDB(ctx, db); err != nil {
		log.Error("migrate.db_unreachable", "error", err)
		os.Exit(1)
	}

	goose.SetBaseFS(migrate.Migrations)
	goose.SetLogger(gooseLogger{log})
	if err := goose.SetDialect("postgres"); err != nil {
		log.Error("migrate.dialect_failed", "error", err)
		os.Exit(1)
	}

	if err := adoptAlembicBaseline(ctx, db, log); err != nil {
		log.Error("migrate.adopt_failed", "error", err)
		os.Exit(1)
	}

	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		log.Error("migrate.up_failed", "error", err)
		os.Exit(1)
	}
	version, _ := goose.GetDBVersionContext(ctx, db)
	log.Info("migrate.done", "version", version)
}

func waitForDB(ctx context.Context, db *sql.DB) error {
	var err error
	for attempt := 0; attempt < 30; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err = db.PingContext(pingCtx)
		cancel()
		if err == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return err
}

// adoptAlembicBaseline — однократное усыновление БД, жившей на Alembic:
// goose-таблицы ещё нет, alembic_version есть → схема уже соответствует
// baseline, помечаем ревизию 1 применённой без выполнения SQL.
func adoptAlembicBaseline(ctx context.Context, db *sql.DB, log *slog.Logger) error {
	var hasGoose, hasAlembic bool
	const existsQ = `SELECT EXISTS (
		SELECT 1 FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_name = $1)`
	if err := db.QueryRowContext(ctx, existsQ, "goose_db_version").Scan(&hasGoose); err != nil {
		return err
	}
	if err := db.QueryRowContext(ctx, existsQ, "alembic_version").Scan(&hasAlembic); err != nil {
		return err
	}
	if hasGoose || !hasAlembic {
		return nil
	}

	// EnsureDBVersion создаёт goose_db_version с нулевой записью.
	if _, err := goose.EnsureDBVersionContext(ctx, db); err != nil {
		return fmt.Errorf("ensure goose table: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO goose_db_version (version_id, is_applied)
		VALUES ($1, TRUE)`, baselineVersion); err != nil {
		return fmt.Errorf("mark baseline: %w", err)
	}
	log.Info("migrate.alembic_adopted", "baseline", baselineVersion)
	return nil
}

type gooseLogger struct{ log *slog.Logger }

func (l gooseLogger) Fatalf(format string, v ...any) {
	l.log.Error(fmt.Sprintf(format, v...))
	os.Exit(1)
}
func (l gooseLogger) Printf(format string, v ...any) {
	l.log.Info(fmt.Sprintf(format, v...))
}
