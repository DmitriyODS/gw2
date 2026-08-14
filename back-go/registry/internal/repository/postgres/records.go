package postgres

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

const recordCols = `id, registry_id, data, created_by, created_at, updated_at`

func scanRecord(row pgx.Row) (*domain.Record, error) {
	var r domain.Record
	err := row.Scan(&r.ID, &r.RegistryID, &r.Data, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if r.Data == nil {
		r.Data = map[string]any{}
	}
	return &r, nil
}

// numericLiteral — что Postgres гарантированно приведёт к numeric. Сортировка
// числового поля обязана пережить мусор в данных: значения записей — JSON, и
// в поле «Количество» человек однажды напишет «уточнить у поставщика». Без
// этого сторожа весь список отвечал 500 (invalid input syntax for type numeric),
// то есть одна кривая ячейка роняла раздел целиком.
const numericLiteral = `^[+-]?([0-9]+([.][0-9]*)?|[.][0-9]+)$`

/*
Построение условий.

	ИНВАРИАНТ: внешние данные в текст запроса не попадают НИКОГДА — ни значения,
	ни идентификаторы полей. Ключ поля в JSONB тоже уезжает параметром
	(data ->> $n), а не подстановкой: Postgres принимает параметр и в пути
	json-оператора, и в выражении ORDER BY. В тексте остаются только номера
	плейсхолдеров, ключевые слова и константы этого файла.

	ph — добавить значение в аргументы и вернуть его плейсхолдер.
*/
func ph(args *[]any, v any) string {
	*args = append(*args, v)
	return "$" + strconv.Itoa(len(*args))
}

// fieldKey — текстовое значение поля записи: data ->> $n. Идентификатор поля
// приезжает аргументом, поэтому строка запроса от него не зависит.
func fieldKey(fieldID int64, args *[]any) string {
	return "data ->> " + ph(args, strconv.FormatInt(fieldID, 10))
}

// numericOf — значение поля как число; нечисловая ячейка даёт NULL, а не роняет
// весь запрос (см. numericLiteral).
func numericOf(key string, args *[]any) string {
	return "CASE WHEN btrim(" + key + ") ~ " + ph(args, numericLiteral) +
		" THEN btrim(" + key + ")::numeric END"
}

// orderBy — выражение сортировки. Направление берётся из белого списка (два
// значения), всё остальное — плейсхолдеры.
func orderBy(f domain.RecordListFilter, args *[]any) string {
	dir := "ASC"
	if f.Desc {
		dir = "DESC"
	}
	if f.SortFieldID <= 0 {
		return "created_at " + dir + ", id " + dir
	}
	key := fieldKey(f.SortFieldID, args)
	switch f.SortKind {
	case "number":
		// Нечисловое значение — NULL: уезжает в конец списка, а не роняет запрос.
		return numericOf(key, args) + " " + dir + " NULLS LAST, id ASC"
	case "date":
		return key + " " + dir + " NULLS LAST, id ASC"
	default:
		return "lower(" + key + ") " + dir + " NULLS LAST, id ASC"
	}
}

// sectionWhere — условие вкладки-подраздела. Одно containment-условие
// покрывает оба вида спискового поля: у одиночного значение — строка
// ("А" @> "А"), у множественного — массив (["А","Б"] @> "А").
func sectionWhere(fieldID int64, value string, args *[]any) string {
	if fieldID <= 0 || value == "" {
		return ""
	}
	return " AND data -> " + ph(args, strconv.FormatInt(fieldID, 10)) +
		" @> to_jsonb(" + ph(args, value) + "::text)"
}

/*
columnWhere — условия фильтров по колонкам таблицы.

	Значение поля лежит в JSONB, поэтому сравнение идёт по data ->> <ключ>. Число
	приводим к numeric только там, где значение к нему приводится: иначе одна
	кривая ячейка роняла бы весь список (та же грабля, что у сортировки).
*/
func columnWhere(filters []domain.ColumnFilter, args *[]any) string {
	var out string
	for _, c := range filters {
		if c.FieldID <= 0 {
			continue
		}
		switch c.Op {
		case "empty":
			key := fieldKey(c.FieldID, args)
			out += " AND (" + key + " IS NULL OR btrim(" + key + ") = '')"
		case "filled":
			key := fieldKey(c.FieldID, args)
			out += " AND " + key + " IS NOT NULL AND btrim(" + key + ") <> ''"
		case "equals":
			if len(c.Values) == 0 {
				continue
			}
			key := fieldKey(c.FieldID, args)
			out += " AND lower(" + key + ") = lower(" + ph(args, c.Values[0]) + ")"
		case "any":
			// Список выбора: подходит любое из отмеченных значений, и поле
			// бывает множественным — потому containment, а не равенство.
			if len(c.Values) == 0 {
				continue
			}
			var parts string
			for _, v := range c.Values {
				if parts != "" {
					parts += " OR "
				}
				parts += "data -> " + ph(args, strconv.FormatInt(c.FieldID, 10)) +
					" @> to_jsonb(" + ph(args, v) + "::text)"
			}
			out += " AND (" + parts + ")"
		case "gt", "lt", "between":
			out += numericRange(c, args)
		default: // contains
			if len(c.Values) == 0 || c.Values[0] == "" {
				continue
			}
			out += " AND " + fieldKey(c.FieldID, args) +
				" ILIKE '%' || " + ph(args, c.Values[0]) + " || '%'"
		}
	}
	return out
}

// numericRange — сравнение «больше/меньше/между» по числовому значению поля.
func numericRange(c domain.ColumnFilter, args *[]any) string {
	if len(c.Values) == 0 {
		return ""
	}
	num := numericOf(fieldKey(c.FieldID, args), args)
	switch c.Op {
	case "gt":
		return " AND " + num + " >= " + ph(args, c.Values[0]) + "::numeric"
	case "lt":
		return " AND " + num + " <= " + ph(args, c.Values[0]) + "::numeric"
	default: // between
		if len(c.Values) < 2 {
			return ""
		}
		return " AND " + num + " BETWEEN " + ph(args, c.Values[0]) + "::numeric AND " +
			ph(args, c.Values[1]) + "::numeric"
	}
}

func (r *Repo) ListRecords(ctx context.Context, f domain.RecordListFilter) ([]*domain.Record, int, error) {
	args := []any{}
	where := `WHERE registry_id = ` + ph(&args, f.RegistryID)
	if f.Search != "" {
		where += ` AND search_text ILIKE '%' || ` + ph(&args, f.Search) + ` || '%'`
	}
	where += sectionWhere(f.SectionFieldID, f.SectionValue, &args)
	where += columnWhere(f.Columns, &args)

	// Счётчик считаем ДО того, как к аргументам добавятся сортировка и границы
	// страницы: у него в запросе их нет, а нумерация плейсхолдеров сквозная.
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM registry_records `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.PerPage
	if limit <= 0 {
		limit = 30
	}
	offset := (f.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}
	order := orderBy(f, &args)
	q := `SELECT ` + recordCols + ` FROM registry_records ` + where +
		` ORDER BY ` + order +
		` LIMIT ` + ph(&args, limit) + ` OFFSET ` + ph(&args, offset)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*domain.Record{}
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, rec)
	}
	return out, total, rows.Err()
}

func (r *Repo) GetRecord(ctx context.Context, id int64) (*domain.Record, error) {
	return scanRecord(r.pool.QueryRow(ctx,
		`SELECT `+recordCols+` FROM registry_records WHERE id = $1`, id))
}

func (r *Repo) CreateRecord(ctx context.Context, rec *domain.Record, searchText string) error {
	return r.pool.QueryRow(ctx,
		`INSERT INTO registry_records (registry_id, data, search_text, created_by)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`,
		rec.RegistryID, rec.Data, searchText, rec.CreatedBy).
		Scan(&rec.ID, &rec.CreatedAt, &rec.UpdatedAt)
}

func (r *Repo) UpdateRecord(ctx context.Context, id int64, data map[string]any, searchText string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE registry_records SET data = $2, search_text = $3, updated_at = now() WHERE id = $1`,
		id, data, searchText)
	return err
}

func (r *Repo) DeleteRecord(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM registry_records WHERE id = $1`, id)
	return err
}

// selectionWhere — условие набора записей под массовую операцию. Явный список
// id бьёт фильтр: человек уже указал, что именно нужно.
func selectionWhere(f domain.ExportFilter, args *[]any) string {
	where := `WHERE registry_id = ` + ph(args, f.RegistryID)
	if len(f.IDs) > 0 {
		return where + ` AND id = ANY(` + ph(args, f.IDs) + `)`
	}
	if f.Search != "" {
		where += ` AND search_text ILIKE '%' || ` + ph(args, f.Search) + ` || '%'`
	}
	where += sectionWhere(f.SectionFieldID, f.SectionValue, args)
	where += columnWhere(f.Columns, args)
	if len(f.Exclude) > 0 {
		where += ` AND NOT (id = ANY(` + ph(args, f.Exclude) + `))`
	}
	return where
}

func (r *Repo) DeleteRecords(ctx context.Context, f domain.ExportFilter) ([]*domain.Record, error) {
	args := []any{}
	where := selectionWhere(f, &args)
	rows, err := r.pool.Query(ctx,
		`DELETE FROM registry_records `+where+` RETURNING `+recordCols, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Record{}
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *Repo) RecordsForExport(ctx context.Context, f domain.ExportFilter) ([]*domain.Record, error) {
	args := []any{}
	where := selectionWhere(f, &args)
	rows, err := r.pool.Query(ctx,
		`SELECT `+recordCols+` FROM registry_records `+where+` ORDER BY created_at DESC, id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Record{}
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *Repo) AllRecords(ctx context.Context, registryID int64) ([]*domain.Record, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+recordCols+` FROM registry_records WHERE registry_id = $1`, registryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Record{}
	for rows.Next() {
		rec, err := scanRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// SearchRecords — глобальный поиск (Hola) по записям ВСЕХ доступных человеку
// реестров одним запросом: свои, расшаренные лично и расшаренные его компаниям.
// search_text поддержан триграммным индексом.
func (r *Repo) SearchRecords(ctx context.Context, userID int64, companyIDs []int64, query string, limit int) ([]*domain.SearchHit, error) {
	if companyIDs == nil {
		companyIDs = []int64{}
	}
	rows, err := r.pool.Query(ctx, `
		SELECT rec.registry_id, reg.name, rec.id, left(rec.search_text, 160)
		FROM registry_records rec
		JOIN registries reg ON reg.id = rec.registry_id
		WHERE rec.search_text ILIKE '%' || $3 || '%'
		  AND (reg.owner_id = $1
		       OR EXISTS (SELECT 1 FROM registry_user_shares sh
		                   WHERE sh.registry_id = reg.id
		                     AND (sh.user_id = $1 OR sh.company_id = ANY($2))))
		ORDER BY rec.id DESC
		LIMIT $4`, userID, companyIDs, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*domain.SearchHit, 0, limit)
	for rows.Next() {
		var h domain.SearchHit
		if err := rows.Scan(&h.RegistryID, &h.RegistryName, &h.RecordID, &h.Snippet); err != nil {
			return nil, err
		}
		out = append(out, &h)
	}
	return out, rows.Err()
}
