package service

import (
	"context"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
)

/* Раздел «Настройки → Хранилище»: биллинг спрашивает владельца файлов, что у
   него ещё живо, и просит удалить выбранное.

   Файлы календаря лежат значениями внутри записей и принадлежат КОМПАНИИ —
   платит её создатель, поэтому работаем по companyIDs (их присылает биллинг,
   он же знает создателей). Удаление очищает поле записи: сама запись со
   всеми остальными значениями и датой остаётся. */

func (s *Service) ListStorageFiles(ctx context.Context, _ int64, companyIDs []int64) ([]storagefiles.File, error) {
	scopes, err := s.repo.EntriesOfCompanies(ctx, companyIDs)
	if err != nil {
		return nil, err
	}
	out := []storagefiles.File{}
	for _, sc := range scopes {
		for _, f := range records.DataFiles(sc.Entry.Data) {
			out = append(out, storagefiles.File{
				Key: f.Path, Name: f.Name, Kind: "entry",
				ID:        strconv.FormatInt(sc.Entry.ID, 10),
				Title:     "Календарь: " + sc.CalendarName,
				CompanyID: sc.CompanyID, CreatedAt: sc.Entry.CreatedAt,
			})
		}
	}
	return out, nil
}

func (s *Service) DeleteStorageFiles(ctx context.Context, _ int64, companyIDs []int64, keys []string) ([]string, error) {
	scopes, err := s.repo.EntriesOfCompanies(ctx, companyIDs)
	if err != nil {
		return nil, err
	}
	drop := make(map[string]bool, len(keys))
	for _, k := range keys {
		drop[k] = true
	}

	deleted := []string{}
	fieldsByCalendar := map[int64][]domain.Field{}
	for _, sc := range scopes {
		data, changed, removed := records.DataWithoutFiles(sc.Entry.Data, drop)
		if !changed {
			continue
		}
		fields, ok := fieldsByCalendar[sc.CalendarID]
		if !ok {
			if fields, err = s.repo.ListFields(ctx, sc.CalendarID); err != nil {
				return nil, err
			}
			fieldsByCalendar[sc.CalendarID] = fields
		}
		// Дату записи не трогаем — правим только значения полей.
		if err := s.repo.UpdateEntry(ctx, sc.Entry.ID, sc.Entry.EventAt, data,
			buildSearchText(fields, data)); err != nil {
			return nil, err
		}
		deleted = append(deleted, removed...)
		sc.Entry.Data = data
		s.bus.Publish(ctx, "entry:updated", []string{roomAll}, entryPayload(sc.CompanyID, sc.Entry))
	}
	if len(deleted) > 0 {
		s.files.Remove(deleted)
	}
	return deleted, nil
}

var _ storagefiles.Owner = (*Service)(nil)
