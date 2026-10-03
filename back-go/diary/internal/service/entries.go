package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
)

const entriesLimit = 2000

// EntryList — выборка записей ежедневника за вкладку (активные/архив).
type EntryList struct {
	Items []*domain.Entry `json:"items"`
}

// ListParams — сырые параметры выборки записей.
type ListParams struct {
	Archived bool
	Search   string
	From     *time.Time
	To       *time.Time
}

// EntryInput — нормализованные поля записи (после разбора тела запроса).
// Нулевая дата — дело без срока, допустимо только в «Моём дне». Attachments
// nil — вложения не трогаем (правка из раздела, который о них не знает).
type EntryInput struct {
	Date        time.Time
	StartMin    *int
	EndMin      *int
	Title       string
	Description string
	Attachments *[]domain.Attachment
}

// day — календарный день значения В ЕГО СОБСТВЕННОЙ зоне (полночь UTC).
// Truncate(24h) здесь не годится: он режет по суткам UTC, из-за чего
// RFC3339-дата с не-UTC смещением («2026-07-05T00:30:00+05:00») уплывала на
// соседний день — клиент прислал 5 июля своей зоны, запись обязана лечь на 5 июля.
func day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// ListEntries — записи ежедневника: активные за диапазон дат (день/неделя/месяц)
// либо весь архив выполненных. Доступно владельцу и адресату (read-only).
func (s *Service) ListEntries(ctx context.Context, userID, diaryID int64, p ListParams) (*EntryList, error) {
	if _, err := s.require(ctx, userID, diaryID, domain.AccessView); err != nil {
		return nil, err
	}
	return s.listEntries(ctx, diaryID, p)
}

func (s *Service) listEntries(ctx context.Context, diaryID int64, p ListParams) (*EntryList, error) {
	// Диапазон дат honored в обоих режимах: активная вкладка передаёт диапазон
	// (просмотр по дню/неделе/месяцу), архив-вкладка — нет (весь архив), а
	// модалка дня тянет выполненные за конкретный день (archived=1 + from/to).
	f := domain.EntryListFilter{
		DiaryID:  diaryID,
		Archived: p.Archived,
		Search:   strings.TrimSpace(p.Search),
		From:     p.From,
		To:       p.To,
		Limit:    entriesLimit,
	}
	items, err := s.repo.ListEntries(ctx, f)
	if err != nil {
		return nil, err
	}
	return &EntryList{Items: items}, nil
}

func (s *Service) GetEntry(ctx context.Context, userID, diaryID, entryID int64) (*domain.Entry, error) {
	if _, err := s.require(ctx, userID, diaryID, domain.AccessView); err != nil {
		return nil, err
	}
	return s.getOwnedEntry(ctx, diaryID, entryID)
}

func (s *Service) getOwnedEntry(ctx context.Context, diaryID, entryID int64) (*domain.Entry, error) {
	e, err := s.repo.GetEntry(ctx, entryID)
	if err != nil {
		return nil, err
	}
	if e == nil || e.DiaryID != diaryID {
		return nil, domain.ErrEntryNotFound
	}
	return e, nil
}

func (s *Service) CreateEntry(ctx context.Context, userID, diaryID int64, in EntryInput) (*domain.Entry, error) {
	d, err := s.require(ctx, userID, diaryID, domain.AccessEdit)
	if err != nil {
		return nil, err
	}
	if err := validateInput(in, d); err != nil {
		return nil, err
	}
	e := &domain.Entry{
		DiaryID: diaryID, Date: dayOf(in.Date),
		StartMin: in.StartMin, EndMin: in.EndMin,
		Title: in.Title, Description: in.Description,
	}
	if in.Attachments != nil {
		if e.Attachments, err = domain.NormalizeAttachments(*in.Attachments); err != nil {
			return nil, err
		}
	}
	if err := s.repo.CreateEntry(ctx, e, searchText(e)); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "diary_entry:created", s.diaryRooms(ctx, d), entryPayload(d.OwnerID, e))
	return e, nil
}

func (s *Service) UpdateEntry(ctx context.Context, userID, diaryID, entryID int64, in EntryInput) (*domain.Entry, error) {
	d, err := s.require(ctx, userID, diaryID, domain.AccessEdit)
	if err != nil {
		return nil, err
	}
	e, err := s.getOwnedEntry(ctx, diaryID, entryID)
	if err != nil {
		return nil, err
	}
	if err := validateInput(in, d); err != nil {
		return nil, err
	}
	if in.Attachments != nil {
		if e.Attachments, err = domain.NormalizeAttachments(*in.Attachments); err != nil {
			return nil, err
		}
	}
	e.Date = dayOf(in.Date)
	e.StartMin, e.EndMin = in.StartMin, in.EndMin
	e.Title, e.Description = in.Title, in.Description
	if err := s.repo.UpdateEntry(ctx, e, searchText(e)); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "diary_entry:updated", s.diaryRooms(ctx, d), entryPayload(d.OwnerID, e))
	return e, nil
}

// SetDone — отметить запись выполненной/невыполненной (перенос в архив и
// обратно). Доступно владельцу и адресату с правом отметки (can_check) —
// сценарий «руководитель раздаёт задачи, сотрудник закрывает».
func (s *Service) SetDone(ctx context.Context, userID, diaryID, entryID int64, done bool) (*domain.Entry, error) {
	d, err := s.require(ctx, userID, diaryID, domain.AccessCheck)
	if err != nil {
		return nil, err
	}
	e, err := s.getOwnedEntry(ctx, diaryID, entryID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetEntryDone(ctx, entryID, done); err != nil {
		return nil, err
	}
	e.Done = done
	s.bus.Publish(ctx, "diary_entry:updated", s.diaryRooms(ctx, d), entryPayload(d.OwnerID, e))
	return e, nil
}

// MoveEntry — перенос записи drag-and-drop'ом: на другой день и/или в другой
// ежедневник (раздача задач по спискам). В обоих ежедневниках нужно право
// вести записи; без срока запись живёт только в «Моём дне».
func (s *Service) MoveEntry(ctx context.Context, userID, diaryID, entryID, targetDiaryID int64, date time.Time) (*domain.Entry, error) {
	d, err := s.require(ctx, userID, diaryID, domain.AccessEdit)
	if err != nil {
		return nil, err
	}
	e, err := s.getOwnedEntry(ctx, diaryID, entryID)
	if err != nil {
		return nil, err
	}
	target := d
	if targetDiaryID != diaryID {
		if target, err = s.require(ctx, userID, targetDiaryID, domain.AccessEdit); err != nil {
			return nil, err
		}
	}
	if date.IsZero() {
		date = e.Date
	}
	dayVal := dayOf(date)
	if dayVal.IsZero() && target.Kind != domain.KindMyDay {
		return nil, domain.ErrDateRequired
	}
	oldDiaryID := e.DiaryID
	if err := s.repo.MoveEntry(ctx, entryID, target.ID, dayVal); err != nil {
		return nil, err
	}
	e.DiaryID, e.Date = target.ID, dayVal
	if target.ID == oldDiaryID {
		s.bus.Publish(ctx, "diary_entry:updated", s.diaryRooms(ctx, d), entryPayload(d.OwnerID, e))
	} else {
		// Перенос между ежедневниками: для подписчиков старого — запись исчезла,
		// для подписчиков нового — появилась.
		s.bus.Publish(ctx, "diary_entry:deleted", s.diaryRooms(ctx, d), map[string]any{
			"id": entryID, "diary_id": oldDiaryID, "owner_id": d.OwnerID,
		})
		s.bus.Publish(ctx, "diary_entry:created", s.diaryRooms(ctx, target), entryPayload(target.OwnerID, e))
	}
	return e, nil
}

// ReorderEntries — ручной порядок записей дня (перетаскивание в модалке дня):
// ids в желаемом порядке получают position 1..N.
func (s *Service) ReorderEntries(ctx context.Context, userID, diaryID int64, date time.Time, ids []int64) error {
	d, err := s.require(ctx, userID, diaryID, domain.AccessEdit)
	if err != nil {
		return err
	}
	if date.IsZero() {
		return domain.ErrDateRequired
	}
	if len(ids) == 0 {
		return nil
	}
	dayVal := day(date)
	if err := s.repo.ReorderEntries(ctx, diaryID, dayVal, ids); err != nil {
		return err
	}
	s.bus.Publish(ctx, "diary_entry:reordered", s.diaryRooms(ctx, d), map[string]any{
		"diary_id": diaryID, "owner_id": d.OwnerID,
		"entry_date": dayVal.Format(domain.DateLayout), "ids": ids,
	})
	return nil
}

// SetLink — привязать/отвязать задачу tasksvc (taskID==nil — отвязать).
func (s *Service) SetLink(ctx context.Context, userID, diaryID, entryID int64, taskID *int64) (*domain.Entry, error) {
	d, err := s.require(ctx, userID, diaryID, domain.AccessEdit)
	if err != nil {
		return nil, err
	}
	e, err := s.getOwnedEntry(ctx, diaryID, entryID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetEntryTask(ctx, entryID, taskID); err != nil {
		return nil, err
	}
	e.LinkedTaskID = taskID
	s.bus.Publish(ctx, "diary_entry:updated", s.diaryRooms(ctx, d), entryPayload(d.OwnerID, e))
	return e, nil
}

func (s *Service) DeleteEntry(ctx context.Context, userID, diaryID, entryID int64) error {
	d, err := s.require(ctx, userID, diaryID, domain.AccessEdit)
	if err != nil {
		return err
	}
	if _, err := s.getOwnedEntry(ctx, diaryID, entryID); err != nil {
		return err
	}
	if err := s.repo.DeleteEntry(ctx, entryID); err != nil {
		return err
	}
	s.bus.Publish(ctx, "diary_entry:deleted", s.diaryRooms(ctx, d), map[string]any{
		"id": entryID, "diary_id": diaryID, "owner_id": d.OwnerID,
	})
	return nil
}

func (s *Service) DeleteEntries(ctx context.Context, userID, diaryID int64, ids []int64) (int64, error) {
	d, err := s.require(ctx, userID, diaryID, domain.AccessEdit)
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	n, err := s.repo.DeleteEntries(ctx, diaryID, ids)
	if err != nil {
		return 0, err
	}
	s.bus.Publish(ctx, "diary_entry:bulk-deleted", s.diaryRooms(ctx, d), map[string]any{
		"ids": ids, "diary_id": diaryID, "owner_id": d.OwnerID,
	})
	return n, nil
}

// dayOf — день записи; нулевое время (без срока) остаётся нулевым.
func dayOf(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return day(t)
}

func validateInput(in EntryInput, d *domain.Diary) error {
	if in.Date.IsZero() && d.Kind != domain.KindMyDay {
		return domain.ErrDateRequired
	}
	if strings.TrimSpace(in.Title) == "" {
		return domain.ErrTitleRequired
	}
	return nil
}

// searchText — строка для сквозного ILIKE-поиска (название + описание).
func searchText(e *domain.Entry) string {
	return strings.ToLower(strings.TrimSpace(e.Title + " " + e.Description))
}

func entryPayload(ownerID int64, e *domain.Entry) map[string]any {
	return map[string]any{
		"id": e.ID, "diary_id": e.DiaryID, "owner_id": ownerID,
		"entry_date": domain.FormatDay(e.Date),
		"start_min":  e.StartMin, "end_min": e.EndMin,
		"title": e.Title, "description": e.Description, "done": e.Done,
		"linked_task_id": e.LinkedTaskID, "attachments": e.Attachments, "position": e.Position,
		"created_at": e.CreatedAt, "updated_at": e.UpdatedAt,
	}
}

// SearchEntries — глобальный поиск по записям доступных ежедневников
// (строка поиска рабочего стола). Пустой запрос ничего не ищет.
func (s *Service) SearchEntries(ctx context.Context, userID int64, query string, limit int) ([]*domain.SearchHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []*domain.SearchHit{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.repo.SearchEntries(ctx, userID, query, limit)
}

// Agenda — невыполненные дела доступных ежедневников за период (живая плитка
// рабочего стола): что осталось сегодня и какое дело ближайшее.
func (s *Service) Agenda(ctx context.Context, userID int64, from, to time.Time, limit int) (*domain.Agenda, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	items, total, err := s.repo.Agenda(ctx, userID, day(from), day(to), limit)
	if err != nil {
		return nil, err
	}
	return &domain.Agenda{Items: items, Total: total}, nil
}

// todayLimit — сколько дел экран «Сегодня» берёт разом: он показывает
// ближайшее, а «Потом» — одной строкой.
const todayLimit = 200

/*
Today — экран «Сегодня» со стороны ежедневников: дела по сегодня включительно
(забытое вчера никуда не девается), дела «Мого дня» без срока и сколько дел
закрыто за день.

	Границы дня присылает клиент — его зоны сервер не знает: dayDate — дата
	дня, from/to — его начало и конец моментами времени.
*/
func (s *Service) Today(ctx context.Context, userID int64, dayDate, from, to time.Time) (*domain.Today, error) {
	my, err := s.repo.MyDay(ctx, userID)
	if err != nil {
		return nil, err
	}
	entries, names, err := s.repo.TodayEntries(ctx, userID, day(dayDate), todayLimit)
	if err != nil {
		return nil, err
	}
	done, err := s.repo.DoneOn(ctx, userID, from, to)
	if err != nil {
		return nil, err
	}
	out := &domain.Today{MyDayID: my.ID, Items: []*domain.Entry{}, Later: []*domain.Entry{},
		Done: done, Diaries: names}
	for _, e := range entries {
		if e.Date.IsZero() {
			out.Later = append(out.Later, e)
		} else {
			out.Items = append(out.Items, e)
		}
	}
	return out, nil
}
