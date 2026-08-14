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
	// Section — значение поля-источника подразделов (вкладка над таблицей);
	// пусто — «Все».
	Section string
	// Columns — фильтры по колонкам таблицы.
	Columns []domain.ColumnFilter
	Page    int
	PerPage int
}

// ListRecords — поиск/фильтры/сортировка/пагинация записей.
func (s *Service) ListRecords(ctx context.Context, userID, registryID int64, p RecordListParams) (*RecordList, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	reg, err := s.require(ctx, a, registryID, domain.AccessView)
	if err != nil {
		return nil, err
	}
	return s.listRecordsByRegistry(ctx, reg, p)
}

// listRecordsByRegistry — ядро выборки страницы записей (без проверки доступа;
// вызывающий уже проверил права или resolveShare). Используется и авторизован-
// ным доступом, и публичным по ссылке.
func (s *Service) listRecordsByRegistry(ctx context.Context, reg *domain.Registry, p RecordListParams) (*RecordList, error) {
	fields, err := s.repo.ListFields(ctx, reg.ID)
	if err != nil {
		return nil, err
	}

	f := domain.RecordListFilter{
		RegistryID: reg.ID,
		Search:     strings.TrimSpace(p.Search),
		Desc:       strings.EqualFold(p.Order, "desc"),
		Columns:    validColumns(fields, p.Columns),
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
	if fid, value := sectionFilter(reg, fields, p.Section); fid > 0 {
		f.SectionFieldID, f.SectionValue = fid, value
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
	if err := s.attachIssues(ctx, reg, items); err != nil {
		return nil, err
	}
	return &RecordList{Items: items, Total: total, Page: f.Page, PerPage: f.PerPage}, nil
}

// validColumns — отсеять фильтры по чужим и несуществующим полям: id колонки
// приходит от клиента, а фильтровать он вправе только по полям ЭТОГО реестра.
func validColumns(fields []domain.Field, in []domain.ColumnFilter) []domain.ColumnFilter {
	out := make([]domain.ColumnFilter, 0, len(in))
	for _, c := range in {
		if findField(fields, c.FieldID) != nil {
			out = append(out, c)
		}
	}
	return out
}

// sectionFilter — перевести вкладку-подраздел в условие по полю-источнику
// (чужое или отключённое поле фильтром не становится). Общий для списка,
// выгрузки и массовых операций — правило одно на всех.
func sectionFilter(reg *domain.Registry, fields []domain.Field, value string) (int64, string) {
	value = strings.TrimSpace(value)
	if value == "" || reg.SectionFieldID == nil {
		return 0, ""
	}
	if f := findField(fields, *reg.SectionFieldID); f != nil && f.Type == domain.FieldSelect {
		return f.ID, value
	}
	return 0, ""
}

// attachIssues — подмешать открытые выдачи учётного реестра: плашка состояния
// нужна всей странице сразу, поэтому один запрос на страницу, а не на запись.
func (s *Service) attachIssues(ctx context.Context, reg *domain.Registry, items []*domain.Record) error {
	if !reg.Accounting || len(items) == 0 {
		return nil
	}
	ids := make([]int64, len(items))
	for i, rec := range items {
		ids[i] = rec.ID
	}
	issues, err := s.repo.OpenIssues(ctx, ids)
	if err != nil {
		return err
	}
	for _, rec := range items {
		rec.Issue = issues[rec.ID]
	}
	return nil
}

func (s *Service) GetRecord(ctx context.Context, userID, registryID, recordID int64) (*domain.Record, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	reg, err := s.require(ctx, a, registryID, domain.AccessView)
	if err != nil {
		return nil, err
	}
	return s.recordIn(ctx, reg, recordID)
}

// recordIn — запись реестра с её открытой выдачей (доступ уже проверен).
func (s *Service) recordIn(ctx context.Context, reg *domain.Registry, recordID int64) (*domain.Record, error) {
	rec, err := s.repo.GetRecord(ctx, recordID)
	if err != nil {
		return nil, err
	}
	if rec == nil || rec.RegistryID != reg.ID {
		return nil, domain.ErrRecordNotFound
	}
	if err := s.attachIssues(ctx, reg, []*domain.Record{rec}); err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *Service) CreateRecord(ctx context.Context, userID, registryID int64, data map[string]any) (*domain.Record, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	reg, err := s.require(ctx, a, registryID, domain.AccessEdit)
	if err != nil {
		return nil, err
	}
	return s.createRecordIn(ctx, reg, &userID, data)
}

func (s *Service) UpdateRecord(ctx context.Context, userID, registryID, recordID int64, data map[string]any) (*domain.Record, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	reg, err := s.require(ctx, a, registryID, domain.AccessEdit)
	if err != nil {
		return nil, err
	}
	return s.updateRecordIn(ctx, reg, recordID, data)
}

// createRecordIn / updateRecordIn — ядро записи БЕЗ проверки доступа: его уже
// сделал вызывающий (участник с уровнем edit либо ссылка того же уровня).
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
	s.publish(ctx, reg.ID, "record:created", recordPayload(rec))
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
	// иначе замена фотографии тихо копила бы мусор в квоте.
	s.removeOrphanFiles(ctx, reg, old, clean)
	rec.Data = clean
	s.publish(ctx, reg.ID, "record:updated", recordPayload(rec))
	return rec, nil
}

func (s *Service) DeleteRecord(ctx context.Context, userID, registryID, recordID int64) error {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return err
	}
	reg, err := s.require(ctx, a, registryID, domain.AccessEdit)
	if err != nil {
		return err
	}
	rec, err := s.recordIn(ctx, reg, recordID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteRecord(ctx, recordID); err != nil {
		return err
	}
	s.removeRecordFiles(ctx, reg, rec)
	s.publish(ctx, registryID, "record:deleted", map[string]any{
		"id": recordID, "registry_id": registryID,
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
	Section string
	Columns []domain.ColumnFilter
	Exclude []int64
}

// DeleteRecords — массовое удаление выбранных записей.
func (s *Service) DeleteRecords(ctx context.Context, userID, registryID int64, p BulkParams) (int64, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return 0, err
	}
	reg, err := s.require(ctx, a, registryID, domain.AccessEdit)
	if err != nil {
		return 0, err
	}
	if !p.All && len(p.IDs) == 0 {
		return 0, nil
	}
	filter, err := s.selectionFilter(ctx, reg, p)
	if err != nil {
		return 0, err
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
	s.removeRecordFiles(ctx, reg, recs...)
	s.publish(ctx, registryID, "record:bulk-deleted", map[string]any{
		"ids": ids, "registry_id": registryID,
	})
	return int64(len(recs)), nil
}

// selectionFilter — набор записей под массовую операцию: явный список id либо
// весь фильтр экрана за вычетом снятых. Общий для удаления, выгрузки и печати
// QR — «выбрано» на всех экранах означает одно и то же.
func (s *Service) selectionFilter(ctx context.Context, reg *domain.Registry, p BulkParams) (domain.ExportFilter, error) {
	filter := domain.ExportFilter{RegistryID: reg.ID}
	if !p.All {
		filter.IDs = p.IDs
		return filter, nil
	}
	fields, err := s.repo.ListFields(ctx, reg.ID)
	if err != nil {
		return filter, err
	}
	filter.Search, filter.Exclude = strings.TrimSpace(p.Search), p.Exclude
	filter.Columns = validColumns(fields, p.Columns)
	filter.SectionFieldID, filter.SectionValue = sectionFilter(reg, fields, p.Section)
	return filter, nil
}

// ── Хелперы ──────────────────────────────────────────────────────

// removeRecordFiles — удалить из хранилища файлы/картинки удаляемых записей.
func (s *Service) removeRecordFiles(ctx context.Context, reg *domain.Registry, recs ...*domain.Record) {
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
		userID, companyID := quotaScope(reg)
		s.files.RemoveFor(ctx, userID, companyID, paths)
	}
}

// removeOrphanFiles — файлы прежних значений, которых нет в новых: замена
// картинки в записи иначе оставляла бы оригинал висеть в квоте.
func (s *Service) removeOrphanFiles(ctx context.Context, reg *domain.Registry, old, next map[string]any) {
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
		userID, companyID := quotaScope(reg)
		s.files.RemoveFor(ctx, userID, companyID, gone)
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
// типу. Неизвестные ключи отбрасываются.
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

func recordPayload(r *domain.Record) map[string]any {
	return map[string]any{
		"id": r.ID, "registry_id": r.RegistryID,
		"data": r.Data, "created_by": r.CreatedBy,
		"created_at": r.CreatedAt, "updated_at": r.UpdatedAt,
	}
}

// SearchRecords — глобальный поиск по записям ВСЕХ доступных реестров (строка
// поиска Hola): свои, расшаренные лично и расшаренные компаниям. Пустой запрос
// ничего не ищет.
func (s *Service) SearchRecords(ctx context.Context, userID int64, query string, limit int) ([]*domain.SearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []*domain.SearchHit{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.SearchRecords(ctx, a.UserID, a.Companies, query, limit)
}
