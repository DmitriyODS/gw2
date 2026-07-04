package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

// RecordList — страница записей реестра.
type RecordList struct {
	Items   []*domain.Record `json:"items"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	PerPage int              `json:"per_page"`
}

// RecordListParams — сырые параметры запроса списка (из query-строки).
type RecordListParams struct {
	Search  string
	Sort    string // "" | "created_at" | "<field_id>"
	Order   string // "asc" | "desc"
	Page    int
	PerPage int
}

// ListRecords — поиск/сортировка/пагинация записей. Сортировка по полю требует
// его типа (для приведения в SQL) — берём из определения реестра.
func (s *Service) ListRecords(ctx context.Context, companyID, registryID int64, p RecordListParams) (*RecordList, error) {
	if _, err := s.requireRegistry(ctx, companyID, registryID); err != nil {
		return nil, err
	}
	return s.listRecordsByRegistry(ctx, registryID, p)
}

// listRecordsByRegistry — ядро выборки страницы записей (без проверки доступа;
// вызывающий уже проверил права или resolveShare). Используется и authed, и
// публичным доступом по ссылке.
func (s *Service) listRecordsByRegistry(ctx context.Context, registryID int64, p RecordListParams) (*RecordList, error) {
	fields, err := s.repo.ListFields(ctx, registryID)
	if err != nil {
		return nil, err
	}

	f := domain.RecordListFilter{
		RegistryID: registryID,
		Search:     strings.TrimSpace(p.Search),
		Desc:       strings.EqualFold(p.Order, "desc"),
		Page:       p.Page,
		PerPage:    p.PerPage,
	}
	if p.Sort != "" && p.Sort != "created_at" {
		if fid, err := strconv.ParseInt(p.Sort, 10, 64); err == nil {
			if field := findField(fields, fid); field != nil {
				f.SortFieldID = fid
				f.SortKind = sortKind(field.Type)
			}
		}
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage <= 0 || f.PerPage > 200 {
		f.PerPage = 30
	}

	items, total, err := s.repo.ListRecords(ctx, f)
	if err != nil {
		return nil, err
	}
	return &RecordList{Items: items, Total: total, Page: f.Page, PerPage: f.PerPage}, nil
}

func (s *Service) GetRecord(ctx context.Context, companyID, registryID, recordID int64) (*domain.Record, error) {
	if _, err := s.requireRegistry(ctx, companyID, registryID); err != nil {
		return nil, err
	}
	rec, err := s.repo.GetRecord(ctx, recordID)
	if err != nil {
		return nil, err
	}
	if rec == nil || rec.RegistryID != registryID {
		return nil, domain.ErrRecordNotFound
	}
	return rec, nil
}

func (s *Service) CreateRecord(ctx context.Context, companyID, registryID, userID int64, data map[string]any) (*domain.Record, error) {
	if _, err := s.requireRegistry(ctx, companyID, registryID); err != nil {
		return nil, err
	}
	fields, err := s.repo.ListFields(ctx, registryID)
	if err != nil {
		return nil, err
	}
	clean, err := coerceData(fields, data)
	if err != nil {
		return nil, err
	}
	rec := &domain.Record{RegistryID: registryID, Data: clean, CreatedBy: &userID}
	if err := s.repo.CreateRecord(ctx, rec, buildSearchText(fields, clean)); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "record:created", []string{roomAll}, recordPayload(companyID, rec))
	return rec, nil
}

func (s *Service) UpdateRecord(ctx context.Context, companyID, registryID, recordID int64, data map[string]any) (*domain.Record, error) {
	rec, err := s.GetRecord(ctx, companyID, registryID, recordID)
	if err != nil {
		return nil, err
	}
	fields, err := s.repo.ListFields(ctx, registryID)
	if err != nil {
		return nil, err
	}
	clean, err := coerceData(fields, data)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateRecord(ctx, recordID, clean, buildSearchText(fields, clean)); err != nil {
		return nil, err
	}
	rec.Data = clean
	s.bus.Publish(ctx, "record:updated", []string{roomAll}, recordPayload(companyID, rec))
	return rec, nil
}

func (s *Service) DeleteRecord(ctx context.Context, companyID, registryID, recordID int64) error {
	rec, err := s.GetRecord(ctx, companyID, registryID, recordID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteRecord(ctx, recordID); err != nil {
		return err
	}
	s.removeRecordFiles(rec)
	s.bus.Publish(ctx, "record:deleted", []string{roomAll}, map[string]any{
		"id": recordID, "registry_id": registryID, "company_id": companyID,
	})
	return nil
}

// DeleteRecords — массовое удаление выбранных записей.
func (s *Service) DeleteRecords(ctx context.Context, companyID, registryID int64, ids []int64) (int64, error) {
	if _, err := s.requireRegistry(ctx, companyID, registryID); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	// Снимаем файлы записей до удаления — после DELETE данные уже недоступны.
	recs, _ := s.repo.RecordsForExport(ctx, registryID, "", ids)
	n, err := s.repo.DeleteRecords(ctx, registryID, ids)
	if err != nil {
		return 0, err
	}
	s.removeRecordFiles(recs...)
	s.bus.Publish(ctx, "record:bulk-deleted", []string{roomAll}, map[string]any{
		"ids": ids, "registry_id": registryID, "company_id": companyID,
	})
	return n, nil
}

// ── Хелперы ──────────────────────────────────────────────────────

// removeRecordFiles — удалить из хранилища файлы/картинки удаляемых записей.
func (s *Service) removeRecordFiles(recs ...*domain.Record) {
	var paths []string
	for _, rec := range recs {
		if rec == nil {
			continue
		}
		for _, v := range rec.Data {
			if p := fileValuePath(v); p != "" {
				paths = append(paths, p)
			}
		}
	}
	if len(paths) > 0 {
		s.files.Remove(paths)
	}
}

// fileValuePath — путь файла/картинки из значения поля. UploadedFile хранится
// как объект с ключом "path"; для прочих типов — пусто.
func fileValuePath(v any) string {
	if m, ok := v.(map[string]any); ok {
		if p, ok := m["path"].(string); ok {
			return p
		}
	}
	return ""
}

func findField(fields []domain.Field, id int64) *domain.Field {
	for i := range fields {
		if fields[i].ID == id {
			return &fields[i]
		}
	}
	return nil
}

func sortKind(fieldType string) string {
	switch fieldType {
	case domain.FieldNumber:
		return "number"
	case domain.FieldDatetime:
		return "date"
	default:
		return "text"
	}
}

// buildSearchText — единая строка для поиска (текст/число/дата/список/ссылка).
func buildSearchText(fields []domain.Field, data map[string]any) string {
	return records.SearchText(fieldInfos(fields), data)
}

// coerceData — оставить только значения определённых полей и проверить их по
// типу (number-маска, варианты select). Неизвестные ключи отбрасываются.
func coerceData(fields []domain.Field, data map[string]any) (map[string]any, error) {
	return records.CoerceData(fieldInfos(fields), data)
}

func fieldInfos(fields []domain.Field) []records.FieldInfo {
	out := make([]records.FieldInfo, len(fields))
	for i, f := range fields {
		out[i] = records.FieldInfo{ID: f.ID, Type: f.Type, Label: f.Label, Config: f.Config}
	}
	return out
}

func recordPayload(companyID int64, r *domain.Record) map[string]any {
	return map[string]any{
		"id": r.ID, "registry_id": r.RegistryID, "company_id": companyID,
		"data": r.Data, "created_by": r.CreatedBy,
		"created_at": r.CreatedAt, "updated_at": r.UpdatedAt,
	}
}
