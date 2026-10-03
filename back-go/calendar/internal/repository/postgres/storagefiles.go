package postgres

import (
	"context"
	"encoding/json"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
)

/* Сторона календарей в контракте владельца файлов (раздел «Хранилище»).

   Файлы лежат значениями внутри calendar_records.data, отдельной таблицы нет.
   За личный календарь платит его хозяин, за календарь команды — создатель
   команды (какие команды создал спрашивающий, знает биллинг). */

func (r *Repo) EntriesForQuota(ctx context.Context, userID int64, companyIDs []int64) ([]*domain.EntryScope, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT rec.id, rec.calendar_id, rec.event_at, rec.data, rec.created_at, rec.updated_at,
		       cal.name, COALESCE(cal.company_id, 0)
		  FROM calendar_records rec
		  JOIN calendars cal ON cal.id = rec.calendar_id
		 WHERE cal.company_id = ANY($1)
		    OR (cal.company_id IS NULL AND cal.owner_id = $2)`, companyIDs, userID)
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
