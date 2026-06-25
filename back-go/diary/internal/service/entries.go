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
type EntryInput struct {
	Date        time.Time
	StartMin    *int
	EndMin      *int
	Title       string
	Description string
}

// ListEntries — записи ежедневника: активные за диапазон дат (день/неделя/месяц)
// либо весь архив выполненных. Доступно владельцу и адресату (read-only).
func (s *Service) ListEntries(ctx context.Context, userID, diaryID int64, p ListParams) (*EntryList, error) {
	if _, _, err := s.requireReadable(ctx, userID, diaryID); err != nil {
		return nil, err
	}
	return s.listEntries(ctx, diaryID, p)
}

func (s *Service) listEntries(ctx context.Context, diaryID int64, p ListParams) (*EntryList, error) {
	f := domain.EntryListFilter{
		DiaryID:  diaryID,
		Archived: p.Archived,
		Search:   strings.TrimSpace(p.Search),
		Limit:    entriesLimit,
	}
	if !p.Archived { // диапазон дат — только для активной вкладки (календарные виды)
		f.From, f.To = p.From, p.To
	}
	items, err := s.repo.ListEntries(ctx, f)
	if err != nil {
		return nil, err
	}
	return &EntryList{Items: items}, nil
}

func (s *Service) GetEntry(ctx context.Context, userID, diaryID, entryID int64) (*domain.Entry, error) {
	if _, _, err := s.requireReadable(ctx, userID, diaryID); err != nil {
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
	d, err := s.requireOwned(ctx, userID, diaryID)
	if err != nil {
		return nil, err
	}
	if err := validateInput(in); err != nil {
		return nil, err
	}
	e := &domain.Entry{
		DiaryID: diaryID, Date: in.Date.Truncate(24 * time.Hour),
		StartMin: in.StartMin, EndMin: in.EndMin,
		Title: in.Title, Description: in.Description,
	}
	if err := s.repo.CreateEntry(ctx, e, searchText(e)); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "diary_entry:created", s.diaryRooms(ctx, d), entryPayload(d.OwnerID, e))
	return e, nil
}

func (s *Service) UpdateEntry(ctx context.Context, userID, diaryID, entryID int64, in EntryInput) (*domain.Entry, error) {
	d, err := s.requireOwned(ctx, userID, diaryID)
	if err != nil {
		return nil, err
	}
	e, err := s.getOwnedEntry(ctx, diaryID, entryID)
	if err != nil {
		return nil, err
	}
	if err := validateInput(in); err != nil {
		return nil, err
	}
	e.Date = in.Date.Truncate(24 * time.Hour)
	e.StartMin, e.EndMin = in.StartMin, in.EndMin
	e.Title, e.Description = in.Title, in.Description
	if err := s.repo.UpdateEntry(ctx, e, searchText(e)); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "diary_entry:updated", s.diaryRooms(ctx, d), entryPayload(d.OwnerID, e))
	return e, nil
}

// SetDone — отметить запись выполненной/невыполненной (перенос в архив и обратно).
func (s *Service) SetDone(ctx context.Context, userID, diaryID, entryID int64, done bool) (*domain.Entry, error) {
	d, err := s.requireOwned(ctx, userID, diaryID)
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

// SetLink — привязать/отвязать задачу tasksvc (taskID==nil — отвязать).
func (s *Service) SetLink(ctx context.Context, userID, diaryID, entryID int64, taskID *int64) (*domain.Entry, error) {
	d, err := s.requireOwned(ctx, userID, diaryID)
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
	d, err := s.requireOwned(ctx, userID, diaryID)
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
		"id": entryID, "diary_id": diaryID, "owner_id": userID,
	})
	return nil
}

func (s *Service) DeleteEntries(ctx context.Context, userID, diaryID int64, ids []int64) (int64, error) {
	d, err := s.requireOwned(ctx, userID, diaryID)
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
		"ids": ids, "diary_id": diaryID, "owner_id": userID,
	})
	return n, nil
}

func validateInput(in EntryInput) error {
	if in.Date.IsZero() {
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
		"entry_date": e.Date.Format(domain.DateLayout),
		"start_min":  e.StartMin, "end_min": e.EndMin,
		"title": e.Title, "description": e.Description, "done": e.Done,
		"linked_task_id": e.LinkedTaskID,
		"created_at":     e.CreatedAt, "updated_at": e.UpdatedAt,
	}
}
