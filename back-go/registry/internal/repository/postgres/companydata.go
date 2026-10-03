package postgres

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/companydata"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

// Перенос компании: устройство раздела совпадает с реестрами и календарями,
// поэтому и выгрузка, и вливание живут общим движком в pkg/records.
var companyTables = records.TableSpec{
	Sets:    "registries",
	Fields:  "registry_fields",
	Records: "registry_records",
	Parent:  "registry_id",
}

func (r *Repo) ExportCompany(ctx context.Context, companyID int64) (companydata.Export, error) {
	return records.ExportCompany(ctx, r.pool, companyTables, companyID)
}

// ImportCompany — влить реестры в команду, созданную под импорт: они лягут в
// её пространство, и участники увидят их с уровнем по умолчанию.
func (r *Repo) ImportCompany(ctx context.Context, in companydata.Import) (int, error) {
	return records.ImportCompany(ctx, r.pool, companyTables, in)
}
