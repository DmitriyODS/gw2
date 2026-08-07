package postgres

import (
	"context"
	"encoding/json"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
)

/* Сторона календарей в контракте владельца файлов (раздел «Хранилище»).

   Файлы лежат значениями внутри calendar_records.data, отдельной таблицы нет.
   Календарь принадлежит КОМПАНИИ, а место за него платит её создатель —
   отсюда отбор по companyIDs (кто создатель, знает биллинг). */

func (r *Repo) EntriesOfCompanies(ctx context.Context, companyIDs []int64) ([]*domain.EntryScope, error) {
	if len(companyIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT rec.id, rec.calendar_id, rec.event_at, rec.data, rec.created_at, rec.updated_at,
		       cal.name, cal.company_id
		  FROM calendar_records rec
		  JOIN calendars cal ON cal.id = rec.calendar_id
		 WHERE cal.company_id = ANY($1)`, companyIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*domain.EntryScope{}
	for rows.Next() {
		var (
			e    domain.Entry
			data []byte
			s    domain.EntryScope
		)
		if err := rows.Scan(&e.ID, &e.CalendarID, &e.EventAt, &data, &e.CreatedAt, &e.UpdatedAt,
			&s.CalendarName, &s.CompanyID); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &e.Data); err != nil {
			continue // битые значения одной записи не должны рушить весь раздел
		}
		s.Entry, s.CalendarID = &e, e.CalendarID
		out = append(out, &s)
	}
	return out, rows.Err()
}
