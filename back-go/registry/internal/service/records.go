package service

import (
	"context"
	"regexp"
	"strconv"
	"strings"

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
	if _, err := s.GetRecord(ctx, companyID, registryID, recordID); err != nil {
		return err
	}
	if err := s.repo.DeleteRecord(ctx, recordID); err != nil {
		return err
	}
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
	n, err := s.repo.DeleteRecords(ctx, registryID, ids)
	if err != nil {
		return 0, err
	}
	s.bus.Publish(ctx, "record:bulk-deleted", []string{roomAll}, map[string]any{
		"ids": ids, "registry_id": registryID, "company_id": companyID,
	})
	return n, nil
}

// ── Хелперы ──────────────────────────────────────────────────────

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
	var b strings.Builder
	for _, f := range fields {
		v, ok := data[domain.FieldID(f.ID)]
		if !ok {
			continue
		}
		if part := domain.SearchContribution(f.Type, v); part != "" {
			b.WriteString(part)
			b.WriteByte(' ')
		}
	}
	return strings.TrimSpace(b.String())
}

// coerceData — оставить только значения определённых полей и проверить их по
// типу (number-маска, варианты select). Неизвестные ключи отбрасываются.
func coerceData(fields []domain.Field, data map[string]any) (map[string]any, error) {
	out := map[string]any{}
	for _, f := range fields {
		key := domain.FieldID(f.ID)
		v, ok := data[key]
		if !ok || v == nil {
			continue
		}
		if err := validateValue(f, v); err != nil {
			return nil, err
		}
		out[key] = v
	}
	return out, nil
}

func validateValue(f domain.Field, v any) error {
	switch f.Type {
	case domain.FieldNumber:
		s := valueString(v)
		if pat := f.NumberPattern(); pat != "" && s != "" {
			re, err := regexp.Compile(pat)
			if err == nil && !re.MatchString(s) {
				return domain.NewError("VALIDATION",
					"Значение поля «"+f.Label+"» не соответствует шаблону", 400)
			}
		}
	case domain.FieldSelect:
		opts := f.FieldOptions()
		if len(opts) == 0 {
			return nil
		}
		allowed := map[string]bool{}
		for _, o := range opts {
			allowed[o] = true
		}
		for _, chosen := range selectValues(v) {
			if !allowed[chosen] {
				return domain.NewError("VALIDATION",
					"Недопустимый вариант поля «"+f.Label+"»", 400)
			}
		}
	}
	return nil
}

func valueString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	switch n := v.(type) {
	case float64:
		return strconv.FormatFloat(n, 'f', -1, 64)
	}
	return ""
}

func selectValues(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		out := make([]string, 0, len(x))
		for _, it := range x {
			if s, ok := it.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func recordPayload(companyID int64, r *domain.Record) map[string]any {
	return map[string]any{
		"id": r.ID, "registry_id": r.RegistryID, "company_id": companyID,
		"data": r.Data, "created_by": r.CreatedBy,
		"created_at": r.CreatedAt, "updated_at": r.UpdatedAt,
	}
}
