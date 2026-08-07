package service

import (
	"sort"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
)

/* Раздел «Настройки → Хранилище»: чем занято место, удаление лишнего и сверка
   с сервисами-владельцами.

   Источник правды о файлах — журнал (billing_storage_files): он пополняется в
   момент заливки, когда имя и размер известны даром. Счётчик
   billing_storage_usage остаётся быстрой суммой для лимитов, а сверка
   приводит оба к тому, что реально лежит у владельцев. */

// TopStorageFiles — сколько крупнейших файлов показываем в разделе.
const TopStorageFiles = 100

// StorageDetails — всё для панели «Хранилище».
type StorageDetails struct {
	LimitBytes int64                  `json:"limit_bytes"` // -1 — без ограничения
	UsedBytes  int64                  `json:"used_bytes"`
	Services   []*domain.StorageEntry `json:"services"`
	Files      []*domain.StoredFile   `json:"files"`
}

func (s *Service) StorageDetails(ctx domain.Ctx, userID int64) (*StorageDetails, error) {
	ent, err := s.Entitlements(ctx, userID, 0)
	if err != nil {
		return nil, err
	}
	services, err := s.Storage.Usage(ctx, userID)
	if err != nil {
		return nil, err
	}
	files, err := s.Storage.TopFiles(ctx, userID, "", TopStorageFiles)
	if err != nil {
		return nil, err
	}
	return &StorageDetails{
		LimitBytes: ent.Limits.StorageBytes,
		UsedBytes:  ent.StorageUsed,
		Services:   services,
		Files:      files,
	}, nil
}

// StorageCleanup — итог операции над хранилищем (удаление или сверка).
type StorageCleanup struct {
	// DeletedFiles/FreedBytes — что ушло по просьбе пользователя или как сироты.
	DeletedFiles int   `json:"deleted_files"`
	FreedBytes   int64 `json:"freed_bytes"`
	// AddedFiles — файлы, которых журнал не знал: они лежат у владельца и
	// теперь учтены (первая сверка подбирает всё, залитое до появления журнала).
	AddedFiles int   `json:"added_files"`
	UsedBytes  int64 `json:"used_bytes"`
}

/* DeleteStorageFiles — удалить выбранные файлы. Ключи группируются по
   сервису-владельцу: снять файл со своей сущности умеет только он (вырезать
   картинку из заметки, снять вложение с сообщения), а нам остаётся журнал и
   счётчик.

   Чужой ключ до владельца не дойдёт: журнал ведётся по владельцу квоты, и в
   выборку попадает только его строка. */
func (s *Service) DeleteStorageFiles(ctx domain.Ctx, userID int64, keys []string) (*StorageCleanup, error) {
	if s.Owners == nil {
		return nil, domain.ErrStorageUnavailable
	}
	if len(keys) == 0 {
		return &StorageCleanup{}, nil
	}
	journal, err := s.Storage.AllFiles(ctx, userID)
	if err != nil {
		return nil, err
	}
	wanted := make(map[string]bool, len(keys))
	for _, k := range keys {
		wanted[k] = true
	}
	byService := map[string][]string{}
	for _, f := range journal {
		if wanted[f.Key] {
			byService[f.Service] = append(byService[f.Service], f.Key)
		}
	}
	if len(byService) == 0 {
		return &StorageCleanup{}, nil
	}

	companyIDs, err := s.Identity.OwnedCompanies(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := &StorageCleanup{}
	for service, serviceKeys := range byService {
		deleted, err := s.Owners.DeleteFiles(ctx, service, userID, companyIDs, serviceKeys)
		if err != nil {
			// Недоступный владелец не должен ронять чистку остальных: его
			// файлы просто останутся на месте.
			s.Log.Warn("storage.delete_failed", "service", service, "error", err)
			continue
		}
		if len(deleted) == 0 {
			continue
		}
		freed, err := s.Storage.RemoveFiles(ctx, userID, deleted)
		if err != nil {
			return nil, err
		}
		if freed > 0 {
			if _, err := s.Storage.Track(ctx, userID, service, -freed); err != nil {
				return nil, err
			}
		}
		out.DeletedFiles += len(deleted)
		out.FreedBytes += freed
	}
	if out.UsedBytes, err = s.Storage.Total(ctx, userID); err != nil {
		return nil, err
	}
	return out, nil
}

/* SweepStorage — сверка журнала с сервисами-владельцами: убрать файлы, за
   которыми уже никто не стоит, доучесть те, о которых журнал не знал, и
   пересчитать счётчик по итогу.

   Это и есть «очистить мусор»: сироты появляются, когда сущность удалили
   мимо учёта (упавшее событие, ручная правка БД), а незнакомые файлы — всё,
   что залито до появления журнала. */
func (s *Service) SweepStorage(ctx domain.Ctx, userID int64) (*StorageCleanup, error) {
	if s.Owners == nil || s.Objects == nil {
		return nil, domain.ErrStorageUnavailable
	}
	companyIDs, err := s.Identity.OwnedCompanies(ctx, userID)
	if err != nil {
		return nil, err
	}
	journal, err := s.Storage.AllFiles(ctx, userID)
	if err != nil {
		return nil, err
	}
	known := make(map[string]*domain.StoredFile, len(journal))
	for _, f := range journal {
		known[f.Key] = f
	}

	out := &StorageCleanup{}
	alive := map[string]bool{}
	for _, service := range s.Owners.Services() {
		files, err := s.Owners.ListFiles(ctx, service, userID, companyIDs)
		if err != nil {
			// Недоступного владельца пропускаем ЦЕЛИКОМ: молчание нельзя
			// принять за «файлов нет», иначе сверка снесёт их все.
			s.Log.Warn("storage.sweep_skipped", "service", service, "error", err)
			for _, f := range journal {
				if f.Service == service {
					alive[f.Key] = true
				}
			}
			continue
		}
		fresh := []*domain.StoredFile{}
		for _, f := range files {
			alive[f.Key] = true
			if prev, ok := known[f.Key]; ok && prev.RefKind == f.RefKind && prev.RefID == f.RefID {
				continue // уже учтён, ссылка не изменилась
			}
			size := int64(0)
			if prev, ok := known[f.Key]; ok {
				size = prev.Size
			} else {
				// Размер знает только хранилище — меряем ровно один раз, при
				// первой встрече с файлом.
				if size, err = s.Objects.Size(ctx, f.Key); err != nil {
					s.Log.Warn("storage.size_failed", "key", f.Key, "error", err)
					continue
				}
				out.AddedFiles++
			}
			fresh = append(fresh, &domain.StoredFile{
				Key: f.Key, Service: service, CompanyID: f.CompanyID, Name: f.Name,
				Size: size, RefKind: f.RefKind, RefID: f.RefID, RefTitle: f.RefTitle,
			})
		}
		if err := s.Storage.AddFiles(ctx, userID, fresh); err != nil {
			return nil, err
		}
	}

	// Сироты: журнал их помнит, а владелец уже нет.
	orphans := []string{}
	for _, f := range journal {
		if !alive[f.Key] {
			orphans = append(orphans, f.Key)
			out.FreedBytes += f.Size
		}
	}
	if len(orphans) > 0 {
		s.Objects.Remove(ctx, orphans...)
		if _, err := s.Storage.RemoveFiles(ctx, userID, orphans); err != nil {
			return nil, err
		}
		out.DeletedFiles = len(orphans)
	}

	// Счётчик приводим к журналу: он и есть итог сверки.
	usage, err := s.Storage.UsageFromFiles(ctx, userID)
	if err != nil {
		return nil, err
	}
	previous, err := s.Storage.Usage(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, entry := range previous {
		if _, ok := usage[entry.Service]; !ok {
			usage[entry.Service] = 0 // раздел опустел — обнуляем, а не бросаем как есть
		}
	}
	for _, service := range sortedKeys(usage) {
		if err := s.Storage.Set(ctx, userID, service, usage[service]); err != nil {
			return nil, err
		}
	}
	if out.UsedBytes, err = s.Storage.Total(ctx, userID); err != nil {
		return nil, err
	}
	return out, nil
}

func sortedKeys(m map[string]int64) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
