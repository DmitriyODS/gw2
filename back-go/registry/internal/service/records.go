package service

import (
	"context"
	"strconv"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

// maxPerPage — потолок размера страницы записей (защита от выгрузки реестра
// целиком одним запросом); клиенты, которым нужны все записи, идут страницами.
const maxPerPage = 200

// RecordList — страница записей реестра.
type RecordList struct {
	Items   []*domain.Record `json:"items"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	PerPage int              `json:"per_page"`
}

// RecordListParams — сырые параметры запроса списка (из query-строки).
type RecordListParams struct {
	Search string
	Sort   string // "" | "created_at" | "<field_id>"
	Order  string // "asc" | "desc"
	// Tag — значение поля-тега реестра (чип над таблицей); пусто — «все».
	Tag     string
	Page    int
	PerPage int
}

// ListRecords — поиск/сортировка/пагинация записей. Сортировка по полю требует
// его типа (для приведения в SQL) — берём из определения реестра.
func (s *Service) ListRecords(ctx context.Context, companyID, registryID int64, p RecordListParams) (*RecordList, error) {
	reg, err := s.requireRegistry(ctx, companyID, registryID)
	if err != nil {
		return nil, err
	}
	return s.listRecordsByRegistry(ctx, reg, p)
}

// listRecordsByRegistry — ядро выборки страницы записей (без проверки доступа;
// вызывающий уже проверил права или resolveShare). Используется и authed, и
// публичным доступом по ссылке.
func (s *Service) listRecordsByRegistry(ctx context.Context, reg *domain.Registry, p RecordListParams) (*RecordList, error) {
	fields, err := s.repo.ListFields(ctx, reg.ID)
	if err != nil {
		return nil, err
	}

	f := domain.RecordListFilter{
		RegistryID: reg.ID,
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
	// Тег фильтрует только по полю, назначенному тегами администратором: чужой
	// или отключённый источник фильтром не становится.
	if tag := strings.TrimSpace(p.Tag); tag != "" && reg.TagFieldID != nil {
		if field := findField(fields, *reg.TagFieldID); field != nil && field.Type == domain.FieldSelect {
			f.TagFieldID = field.ID
			f.TagValue = tag
		}
	}
	if f.Page < 1 {
		f.Page = 1
	}
	// Просьбу о слишком большой странице ЗАЖИМАЕМ до максимума, а не сбрасываем
	// в дефолт: иначе клиент, которому нужны все записи, молча получал 30 первых
	// и считал, что остальных не существует.
	if f.PerPage <= 0 {
		f.PerPage = 30
	}
	if f.PerPage > maxPerPage {
		f.PerPage = maxPerPage
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
	reg, err := s.requireRegistry(ctx, companyID, registryID)
	if err != nil {
		return nil, err
	}
	return s.createRecordIn(ctx, reg, &userID, data)
}

func (s *Service) UpdateRecord(ctx context.Context, companyID, registryID, recordID int64, data map[string]any) (*domain.Record, error) {
	reg, err := s.requireRegistry(ctx, companyID, registryID)
	if err != nil {
		return nil, err
	}
	return s.updateRecordIn(ctx, reg, recordID, data)
}

// createRecordIn / updateRecordIn — ядро записи БЕЗ проверки доступа: его уже
// сделал вызывающий (участник компании либо ссылка уровня edit). Компания для
// событий берётся у самого реестра — по коду ссылки её взять больше неоткуда.
func (s *Service) createRecordIn(ctx context.Context, reg *domain.Registry, createdBy *int64, data map[string]any) (*domain.Record, error) {
	fields, err := s.repo.ListFields(ctx, reg.ID)
	if err != nil {
		return nil, err
	}
	clean, err := coerceData(fields, data)
	if err != nil {
		return nil, err
	}
	rec := &domain.Record{RegistryID: reg.ID, Data: clean, CreatedBy: createdBy}
	if err := s.repo.CreateRecord(ctx, rec, buildSearchText(fields, clean)); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "record:created", []string{roomAll}, recordPayload(reg.CompanyID, rec))
	return rec, nil
}

func (s *Service) updateRecordIn(ctx context.Context, reg *domain.Registry, recordID int64, data map[string]any) (*domain.Record, error) {
	rec, err := s.repo.GetRecord(ctx, recordID)
	if err != nil {
		return nil, err
	}
	if rec == nil || rec.RegistryID != reg.ID {
		return nil, domain.ErrRecordNotFound
	}
	fields, err := s.repo.ListFields(ctx, reg.ID)
	if err != nil {
		return nil, err
	}
	clean, err := coerceData(fields, data)
	if err != nil {
		return nil, err
	}
	// Прежние значения запоминаем ДО записи: репозиторий волен обновить снимок.
	old := rec.Data
	if err := s.repo.UpdateRecord(ctx, recordID, clean, buildSearchText(fields, clean)); err != nil {
		return nil, err
	}
	// Файлы и картинки, оставшиеся не у дел после правки, из хранилища убираем:
	// иначе замена фотографии тихо копила бы мусор в квоте компании.
	s.removeOrphanFiles(ctx, reg.CompanyID, old, clean)
	rec.Data = clean
	s.bus.Publish(ctx, "record:updated", []string{roomAll}, recordPayload(reg.CompanyID, rec))
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
	s.removeRecordFiles(ctx, companyID, rec)
	s.bus.Publish(ctx, "record:deleted", []string{roomAll}, map[string]any{
		"id": recordID, "registry_id": registryID, "company_id": companyID,
	})
	return nil
}

// BulkParams — что удаляем: перечисленные записи либо ВЕСЬ текущий фильтр
// экрана за вычетом снятых галочек (выбор «отметить всё» живёт между
// страницами, поэтому приходит фильтром, а не списком id).
type BulkParams struct {
	IDs     []int64
	All     bool
	Search  string
	Tag     string
	Exclude []int64
}

// DeleteRecords — массовое удаление выбранных записей.
func (s *Service) DeleteRecords(ctx context.Context, companyID, registryID int64, p BulkParams) (int64, error) {
	reg, err := s.requireRegistry(ctx, companyID, registryID)
	if err != nil {
		return 0, err
	}
	if !p.All && len(p.IDs) == 0 {
		return 0, nil
	}
	filter := domain.ExportFilter{RegistryID: registryID}
	if p.All {
		filter.Search, filter.Exclude = strings.TrimSpace(p.Search), p.Exclude
		s.applyTagFilter(ctx, reg, p.Tag, &filter)
	} else {
		filter.IDs = p.IDs
	}
	// Удаление возвращает сами записи: id — событию, data — чистке файлов.
	recs, err := s.repo.DeleteRecords(ctx, filter)
	if err != nil {
		return 0, err
	}
	if len(recs) == 0 {
		return 0, nil
	}
	ids := make([]int64, 0, len(recs))
	for _, rec := range recs {
		ids = append(ids, rec.ID)
	}
	s.removeRecordFiles(ctx, companyID, recs...)
	s.bus.Publish(ctx, "record:bulk-deleted", []string{roomAll}, map[string]any{
		"ids": ids, "registry_id": registryID, "company_id": companyID,
	})
	return int64(len(recs)), nil
}

// applyTagFilter — перевести чип-тег в условие по полю-источнику тегов реестра
// (чужое или отключённое поле фильтром не становится). Общий для списка,
// выгрузки и массового удаления — правило одно на всех.
func (s *Service) applyTagFilter(ctx context.Context, reg *domain.Registry, tag string, out *domain.ExportFilter) {
	tag = strings.TrimSpace(tag)
	if tag == "" || reg.TagFieldID == nil {
		return
	}
	fields, err := s.repo.ListFields(ctx, reg.ID)
	if err != nil {
		return
	}
	if f := findField(fields, *reg.TagFieldID); f != nil && f.Type == domain.FieldSelect {
		out.TagFieldID, out.TagValue = f.ID, tag
	}
}

// ── Хелперы ──────────────────────────────────────────────────────

// removeRecordFiles — удалить из хранилища файлы/картинки удаляемых записей.
func (s *Service) removeRecordFiles(ctx context.Context, companyID int64, recs ...*domain.Record) {
	var paths []string
	for _, rec := range recs {
		if rec == nil {
			continue
		}
		for _, v := range rec.Data {
			paths = append(paths, filePaths(v)...)
		}
	}
	if len(paths) > 0 {
		s.files.RemoveFor(ctx, 0, companyID, paths)
	}
}

// removeOrphanFiles — файлы прежних значений, которых нет в новых: замена
// картинки в записи иначе оставляла бы оригинал висеть в квоте компании.
func (s *Service) removeOrphanFiles(ctx context.Context, companyID int64, old, next map[string]any) {
	kept := map[string]bool{}
	for _, v := range next {
		for _, p := range filePaths(v) {
			kept[p] = true
		}
	}
	var gone []string
	for _, v := range old {
		for _, p := range filePaths(v) {
			if !kept[p] {
				gone = append(gone, p)
			}
		}
	}
	if len(gone) > 0 {
		s.files.RemoveFor(ctx, 0, companyID, gone)
	}
}

// filePaths — ключи хранилища из значения поля: сам файл и (у картинок) его
// миниатюра. UploadedFile хранится объектом с ключами "path"/"thumb"; для
// прочих типов — пусто.
func filePaths(v any) []string {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	var out []string
	if p, ok := m["path"].(string); ok && p != "" {
		out = append(out, p)
	}
	if t, ok := m["thumb"].(string); ok && t != "" {
		out = append(out, t)
	}
	return out
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

// SearchRecords — глобальный поиск по записям всех реестров компании (строка
// поиска рабочего стола). Пустой запрос ничего не ищет.
func (s *Service) SearchRecords(ctx context.Context, companyID int64, query string, limit int) ([]*domain.SearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []*domain.SearchHit{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.repo.SearchRecords(ctx, companyID, query, limit)
}
