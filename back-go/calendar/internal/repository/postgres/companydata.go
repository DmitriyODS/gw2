package postgres

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/companydata"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

// Перенос компании: устройство раздела совпадает с реестрами и календарями,
// поэтому и выгрузка, и вливание живут общим движком в pkg/records.
var companyTables = records.TableSpec{
	Sets:    "calendars",
	Fields:  "calendar_fields",
	Records: "calendar_records",
	Parent:  "calendar_id",
}

func (r *Repo) ExportCompany(ctx context.Context, companyID int64) (companydata.Export, error) {
	return records.ExportCompany(ctx, r.pool, companyTables, companyID)
}

func (r *Repo) ImportCompany(ctx context.Context, in companydata.Import) (int, error) {
	return records.ImportCompany(ctx, r.pool, companyTables, in)
}
