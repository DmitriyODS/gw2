package postgres

import (
	"context"
	"encoding/json"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

/* Сторона реестров в контракте владельца файлов (раздел «Хранилище»).

   Файлы лежат значениями внутри registry_records.data, отдельной таблицы нет.
   Реестр принадлежит КОМПАНИИ, а место за него платит её создатель — отсюда
   отбор по companyIDs (кто создатель, знает биллинг). */

func (r *Repo) RecordsOfCompanies(ctx context.Context, companyIDs []int64) ([]*domain.RecordScope, error) {
	if len(companyIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT rec.id, rec.registry_id, rec.data, rec.created_at, rec.updated_at,
		       reg.name, reg.company_id
		  FROM registry_records rec
		  JOIN registries reg ON reg.id = rec.registry_id
		 WHERE reg.company_id = ANY($1)`, companyIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*domain.RecordScope{}
	for rows.Next() {
		var (
			rec  domain.Record
			data []byte
			s    domain.RecordScope
		)
		if err := rows.Scan(&rec.ID, &rec.RegistryID, &data, &rec.CreatedAt, &rec.UpdatedAt,
			&s.RegistryName, &s.CompanyID); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &rec.Data); err != nil {
			continue // битые значения одной записи не должны рушить весь раздел
		}
		s.Record, s.RegistryID = &rec, rec.RegistryID
		out = append(out, &s)
	}
	return out, rows.Err()
}
