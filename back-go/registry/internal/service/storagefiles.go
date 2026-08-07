package service

import (
	"context"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

/* Раздел «Настройки → Хранилище»: биллинг спрашивает владельца файлов, что у
   него ещё живо, и просит удалить выбранное.

   Файлы реестра лежат значениями внутри записей и принадлежат КОМПАНИИ —
   платит её создатель, поэтому работаем по companyIDs (их присылает биллинг,
   он же знает создателей). Удаление очищает поле записи: сама запись со
   всеми остальными значениями остаётся. */

func (s *Service) ListStorageFiles(ctx context.Context, _ int64, companyIDs []int64) ([]storagefiles.File, error) {
	scopes, err := s.repo.RecordsOfCompanies(ctx, companyIDs)
	if err != nil {
		return nil, err
	}
	out := []storagefiles.File{}
	for _, sc := range scopes {
		for _, f := range records.DataFiles(sc.Record.Data) {
			out = append(out, storagefiles.File{
				Key: f.Path, Name: f.Name, Kind: "record",
				ID:        strconv.FormatInt(sc.Record.ID, 10),
				Title:     "Реестр: " + sc.RegistryName,
				CompanyID: sc.CompanyID, CreatedAt: sc.Record.CreatedAt,
			})
		}
	}
	return out, nil
}

func (s *Service) DeleteStorageFiles(ctx context.Context, _ int64, companyIDs []int64, keys []string) ([]string, error) {
	scopes, err := s.repo.RecordsOfCompanies(ctx, companyIDs)
	if err != nil {
		return nil, err
	}
	drop := make(map[string]bool, len(keys))
	for _, k := range keys {
		drop[k] = true
	}

	deleted := []string{}
	fieldsByRegistry := map[int64][]domain.Field{}
	for _, sc := range scopes {
		data, changed, removed := records.DataWithoutFiles(sc.Record.Data, drop)
		if !changed {
			continue
		}
		fields, ok := fieldsByRegistry[sc.RegistryID]
		if !ok {
			if fields, err = s.repo.ListFields(ctx, sc.RegistryID); err != nil {
				return nil, err
			}
			fieldsByRegistry[sc.RegistryID] = fields
		}
		if err := s.repo.UpdateRecord(ctx, sc.Record.ID, data, buildSearchText(fields, data)); err != nil {
			return nil, err
		}
		deleted = append(deleted, removed...)
		sc.Record.Data = data
		s.bus.Publish(ctx, "record:updated", []string{roomAll}, recordPayload(sc.CompanyID, sc.Record))
	}
	if len(deleted) > 0 {
		s.files.Remove(deleted)
	}
	return deleted, nil
}

var _ storagefiles.Owner = (*Service)(nil)
