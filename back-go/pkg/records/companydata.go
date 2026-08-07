package records

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/pkg/companydata"
)

/* Перенос компании для «настраиваемых записей» — реестров и календарей.
   Устроены они одинаково (набор → поля → записи), поэтому движок здесь один,
   а сервис задаёт лишь имена своих таблиц.

   Строки едут СЫРЫМ jsonb (`to_jsonb`), а не разобранными структурами: колонки
   у наборов разные (у календаря есть event_at и условная видимость), и
   перечислять их значило бы править перенос при каждой новой колонке.
   Идентификаторы выдаёт последовательность ДО вставки — так ссылки внутри
   архива (поле → набор, ключи data → id поля) переписываются одним проходом. */

// TableSpec — имена таблиц раздела.
type TableSpec struct {
	Sets    string // registries | calendars
	Fields  string // registry_fields | calendar_fields
	Records string // registry_records | calendar_records
	Parent  string // registry_id | calendar_id — ссылка полей и записей на набор
}

type row = map[string]any

type companyDump struct {
	Sets    []row `json:"sets"`
	Fields  []row `json:"fields"`
	Records []row `json:"records"`
}

// ExportCompany — наборы, их поля и записи компании одним JSON.
func ExportCompany(ctx context.Context, pool *pgxpool.Pool, spec TableSpec, companyID int64) (companydata.Export, error) {
	sets, err := dumpRows(ctx, pool,
		fmt.Sprintf(`SELECT to_jsonb(t) FROM %s t WHERE company_id = $1 ORDER BY id`, spec.Sets), companyID)
	if err != nil {
		return companydata.Export{}, err
	}
	ids := idsOf(sets)
	fields, err := dumpRows(ctx, pool,
		fmt.Sprintf(`SELECT to_jsonb(t) FROM %s t WHERE %s = ANY($1) ORDER BY id`, spec.Fields, spec.Parent), ids)
	if err != nil {
		return companydata.Export{}, err
	}
	recs, err := dumpRows(ctx, pool,
		fmt.Sprintf(`SELECT to_jsonb(t) FROM %s t WHERE %s = ANY($1) ORDER BY id`, spec.Records, spec.Parent), ids)
	if err != nil {
		return companydata.Export{}, err
	}

	payload, err := json.Marshal(companyDump{Sets: sets, Fields: fields, Records: recs})
	if err != nil {
		return companydata.Export{}, err
	}
	keys := []string{}
	for _, r := range recs {
		for _, f := range dataFilesOf(r) {
			keys = append(keys, f)
		}
	}
	return companydata.Export{Payload: payload, FileKeys: keys, Count: len(recs)}, nil
}

// ImportCompany — влить наборы в компанию, созданную под импорт.
func ImportCompany(ctx context.Context, pool *pgxpool.Pool, spec TableSpec, in companydata.Import) (int, error) {
	var dump companyDump
	if err := json.Unmarshal(in.Payload, &dump); err != nil {
		return 0, err
	}
	if len(dump.Sets) == 0 {
		return 0, nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	setIDs, err := reserveIDs(ctx, tx, spec.Sets, len(dump.Sets))
	if err != nil {
		return 0, err
	}
	sets := map[int64]int64{}
	for i, r := range dump.Sets {
		sets[intOf(r["id"])] = setIDs[i]
		r["id"] = setIDs[i]
		r["company_id"] = in.CompanyID
		remapUser(r, "created_by", in)
	}

	fieldIDs, err := reserveIDs(ctx, tx, spec.Fields, len(dump.Fields))
	if err != nil {
		return 0, err
	}
	fields := map[int64]int64{}
	kept := make([]row, 0, len(dump.Fields))
	for i, r := range dump.Fields {
		parent, ok := sets[intOf(r[spec.Parent])]
		if !ok {
			continue
		}
		fields[intOf(r["id"])] = fieldIDs[i]
		r["id"] = fieldIDs[i]
		r[spec.Parent] = parent
		kept = append(kept, r)
	}
	dump.Fields = kept
	// Условная видимость ссылается на другое поле — переписываем после того,
	// как известны все новые идентификаторы.
	for _, r := range dump.Fields {
		if v, ok := r["visible_field_id"]; ok && v != nil {
			if id, ok := fields[intOf(v)]; ok {
				r["visible_field_id"] = id
			} else {
				r["visible_field_id"] = nil
				r["visible_value"] = nil
			}
		}
	}

	recIDs, err := reserveIDs(ctx, tx, spec.Records, len(dump.Records))
	if err != nil {
		return 0, err
	}
	keptRecs := make([]row, 0, len(dump.Records))
	for i, r := range dump.Records {
		parent, ok := sets[intOf(r[spec.Parent])]
		if !ok {
			continue
		}
		r["id"] = recIDs[i]
		r[spec.Parent] = parent
		remapUser(r, "created_by", in)
		r["data"] = remapData(r["data"], fields, in)
		keptRecs = append(keptRecs, r)
	}
	dump.Records = keptRecs

	for _, part := range []struct {
		table string
		rows  []row
	}{{spec.Sets, dump.Sets}, {spec.Fields, dump.Fields}, {spec.Records, dump.Records}} {
		if err := insertRows(ctx, tx, part.table, part.rows); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(dump.Records), nil
}

/* ── Вспомогательное ───────────────────────────────────────────── */

func dumpRows(ctx context.Context, pool *pgxpool.Pool, query string, arg any) ([]row, error) {
	rows, err := pool.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []row{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var r row
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func idsOf(rows []row) []int64 {
	out := make([]int64, 0, len(rows))
	for _, r := range rows {
		out = append(out, intOf(r["id"]))
	}
	return out
}

// reserveIDs — новые идентификаторы пачкой из последовательности таблицы:
// ссылки внутри архива переписываются до вставки, а не после неё.
func reserveIDs(ctx context.Context, tx pgx.Tx, table string, n int) ([]int64, error) {
	if n == 0 {
		return nil, nil
	}
	rows, err := tx.Query(ctx,
		`SELECT nextval(pg_get_serial_sequence($1, 'id')) FROM generate_series(1, $2)`, table, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]int64, 0, n)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// insertRows — вставка сырых строк. Колонки, которых в целевой схеме нет,
// jsonb_populate_recordset отбрасывает сам — архив со старой версии не падает.
func insertRows(ctx context.Context, tx pgx.Tx, table string, rows []row) error {
	if len(rows) == 0 {
		return nil
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		fmt.Sprintf(`INSERT INTO %s SELECT * FROM jsonb_populate_recordset(NULL::%s, $1::jsonb)`, table, table),
		string(payload))
	return err
}

func intOf(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case json.Number:
		i, _ := n.Int64()
		return i
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	}
	return 0
}

func remapUser(r row, key string, in companydata.Import) {
	if v, ok := r[key]; ok && v != nil {
		r[key] = in.UserID(intOf(v))
	}
}

// remapData — значения записи: ключи это строковые id полей, а внутри могут
// лежать файлы, уже перенесённые в хранилище под новыми ключами.
func remapData(v any, fields map[int64]int64, in companydata.Import) any {
	data, ok := v.(map[string]any)
	if !ok {
		return v
	}
	out := make(map[string]any, len(data))
	for key, val := range data {
		id, ok := fields[intOf(key)]
		if !ok {
			continue // поле не доехало — значение без него не имеет смысла
		}
		if f, isFile := val.(map[string]any); isFile {
			if path, ok := f["path"].(string); ok && path != "" {
				f["path"] = in.FileKey(path)
			}
		}
		out[strconv.FormatInt(id, 10)] = val
	}
	return out
}

func dataFilesOf(r row) []string {
	data, ok := r["data"].(map[string]any)
	if !ok {
		return nil
	}
	out := []string{}
	for _, f := range DataFiles(data) {
		if f.Path != "" {
			out = append(out, f.Path)
		}
	}
	return out
}
