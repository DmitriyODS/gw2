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

// ImportCompany — влить реестры в компанию, созданную под импорт, и раздать их
// ей же. Видимость реестра держит шара, а не колонка company_id: без этой
// строки перенесённые реестры увидел бы один их владелец, и компания приехала
// бы пустой.
func (r *Repo) ImportCompany(ctx context.Context, in companydata.Import) (int, error) {
	n, err := records.ImportCompany(ctx, r.pool, companyTables, in)
	if err != nil || n == 0 {
		return n, err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO registry_user_shares (registry_id, company_id, access, created_by)
		SELECT id, company_id, 'edit', owner_id FROM registries WHERE company_id = $1
		ON CONFLICT (registry_id, company_id) WHERE company_id IS NOT NULL DO NOTHING`,
		in.CompanyID)
	return n, err
}
