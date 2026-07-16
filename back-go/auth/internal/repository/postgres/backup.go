package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// BackupStore — универсальный дамп/восстановление таблиц общей БД. Все сервисы
// делят одну PostgreSQL, поэтому authsvc читает и пишет таблицы всех разделов.
// Дамп схемо-независим: состав колонок берётся из to_jsonb/jsonb_populate_recordset,
// благодаря чему новые таблицы попадают в бэкап без правок кода.
type BackupStore struct {
	pool *pgxpool.Pool
}

func NewBackupStore(pool *pgxpool.Pool) *BackupStore {
	return &BackupStore{pool: pool}
}

var _ domain.BackupStore = (*BackupStore)(nil)

// AllTables — все обычные таблицы public-схемы минус исключённые.
func (b *BackupStore) AllTables(ctx context.Context) ([]string, error) {
	rows, err := b.pool.Query(ctx, `
		SELECT table_name FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		 ORDER BY table_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		if domain.BackupExcluded[t] {
			continue
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ExportTables — для каждой таблицы выгружает все строки одним JSON-массивом.
func (b *BackupStore) ExportTables(ctx context.Context, tables []string) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(tables))
	for _, t := range tables {
		ident := pgx.Identifier{t}.Sanitize()
		var raw []byte
		err := b.pool.QueryRow(ctx,
			fmt.Sprintf(`SELECT coalesce(jsonb_agg(to_jsonb(x)), '[]'::jsonb) FROM %s x`, ident)).
			Scan(&raw)
		if err != nil {
			return nil, fmt.Errorf("export %s: %w", t, err)
		}
		out[t] = json.RawMessage(raw)
	}
	return out, nil
}

// ImportTables — ДЕСТРУКТИВНОЕ восстановление в одной транзакции: TRUNCATE ...
// RESTART IDENTITY CASCADE для выбранных таблиц, затем вставки в FK-безопасном
// порядке и setval серийных последовательностей. jsonb_populate_recordset
// раскрывает строки в записи таблицы — типы восстанавливает сам PostgreSQL,
// само-ссылки внутри одной таблицы валидны (FK проверяется в конце стейтмента).
func (b *BackupStore) ImportTables(ctx context.Context, tables []string, data map[string]json.RawMessage) error {
	if len(tables) == 0 {
		return nil
	}

	ordered, err := b.orderByFK(ctx, tables)
	if err != nil {
		return err
	}

	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	idents := make([]string, len(ordered))
	for i, t := range ordered {
		idents[i] = pgx.Identifier{t}.Sanitize()
	}
	if _, err := tx.Exec(ctx,
		`TRUNCATE `+strings.Join(idents, ", ")+` RESTART IDENTITY CASCADE`); err != nil {
		return err
	}

	for _, t := range ordered {
		raw := data[t]
		if len(raw) == 0 || string(raw) == "[]" {
			continue
		}
		ident := pgx.Identifier{t}.Sanitize()
		if _, err := tx.Exec(ctx, fmt.Sprintf(
			`INSERT INTO %s SELECT * FROM jsonb_populate_recordset(NULL::%s, $1::jsonb)`,
			ident, ident), []byte(raw)); err != nil {
			return fmt.Errorf("import %s: %w", t, err)
		}
	}

	// Серийные последовательности (колонка id) — за макс. id. Таблицы БЕЗ
	// колонки id (составной PK: device_tokens, task_tags, portal_seen…) сюда
	// не попадают: pg_get_serial_sequence на них падает с 42703 и уронил бы
	// весь импорт — он идёт одной транзакцией.
	withID, err := b.tablesWithIDColumn(ctx, tx, ordered)
	if err != nil {
		return err
	}
	for _, t := range withID {
		var seq *string
		if err := tx.QueryRow(ctx, `SELECT pg_get_serial_sequence($1, 'id')`, t).Scan(&seq); err != nil {
			return err
		}
		if seq == nil || *seq == "" {
			continue // колонка id есть, но не серийная (id задаётся вручную)
		}
		ident := pgx.Identifier{t}.Sanitize()
		if _, err := tx.Exec(ctx, fmt.Sprintf(
			`SELECT setval($1, GREATEST((SELECT COALESCE(MAX(id), 1) FROM %s), 1))`, ident),
			*seq); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// tablesWithIDColumn — какие из таблиц вообще имеют колонку id (только у них
// бывает серийная последовательность). Порядок сохраняется.
func (b *BackupStore) tablesWithIDColumn(ctx context.Context, tx pgx.Tx, tables []string) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT table_name FROM information_schema.columns
		WHERE table_schema = 'public' AND column_name = 'id' AND table_name = ANY($1)`, tables)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	has := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		has[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(has))
	for _, t := range tables {
		if has[t] {
			out = append(out, t)
		}
	}
	return out, nil
}

// orderByFK — топологическая сортировка таблиц по внешним ключам (родитель
// раньше ребёнка) в пределах переданного набора. Само-ссылки игнорируются.
func (b *BackupStore) orderByFK(ctx context.Context, tables []string) ([]string, error) {
	set := make(map[string]bool, len(tables))
	for _, t := range tables {
		set[t] = true
	}

	rows, err := b.pool.Query(ctx, `
		SELECT DISTINCT tc.table_name AS child, ccu.table_name AS parent
		  FROM information_schema.table_constraints tc
		  JOIN information_schema.constraint_column_usage ccu
		    ON ccu.constraint_name = tc.constraint_name
		   AND ccu.table_schema = tc.table_schema
		 WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_schema = 'public'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deps := make(map[string]map[string]bool, len(tables)) // child → parents
	indeg := make(map[string]int, len(tables))
	for t := range set {
		deps[t] = map[string]bool{}
	}
	for rows.Next() {
		var child, parent string
		if err := rows.Scan(&child, &parent); err != nil {
			return nil, err
		}
		if child == parent || !set[child] || !set[parent] {
			continue
		}
		if !deps[child][parent] {
			deps[child][parent] = true
			indeg[child]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Kahn: устойчивый порядок — обрабатываем в исходном порядке tables.
	out := make([]string, 0, len(tables))
	placed := map[string]bool{}
	remaining := append([]string(nil), tables...)
	for len(out) < len(tables) {
		progress := false
		next := remaining[:0]
		for _, t := range remaining {
			if placed[t] {
				continue
			}
			ready := true
			for parent := range deps[t] {
				if !placed[parent] {
					ready = false
					break
				}
			}
			if ready {
				out = append(out, t)
				placed[t] = true
				progress = true
			} else {
				next = append(next, t)
			}
		}
		remaining = next
		if !progress {
			// Цикл (не должно быть) — добавляем остаток как есть.
			for _, t := range remaining {
				if !placed[t] {
					out = append(out, t)
					placed[t] = true
				}
			}
			break
		}
	}
	return out, nil
}
