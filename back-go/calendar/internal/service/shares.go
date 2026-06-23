package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/DmitriyODS/gw2/back-go/calendar/internal/domain"
)

var errShareNotFound = domain.NewError("NOT_FOUND", "Ссылка не найдена или отозвана", 404)

// ── Управление ссылками (требует прав участника компании) ──

func (s *Service) ListShares(ctx context.Context, companyID, calendarID int64) ([]*domain.Share, error) {
	if _, err := s.requireCalendar(ctx, companyID, calendarID); err != nil {
		return nil, err
	}
	return s.repo.ListShares(ctx, calendarID)
}

func (s *Service) CreateShare(ctx context.Context, companyID, calendarID, userID int64) (*domain.Share, error) {
	if _, err := s.requireCalendar(ctx, companyID, calendarID); err != nil {
		return nil, err
	}
	code, err := newShareCode()
	if err != nil {
		return nil, err
	}
	share := &domain.Share{CalendarID: calendarID, Code: code, CreatedBy: &userID}
	if err := s.repo.CreateShare(ctx, share); err != nil {
		return nil, err
	}
	return share, nil
}

func (s *Service) RevokeShare(ctx context.Context, companyID, calendarID, shareID int64) error {
	if _, err := s.requireCalendar(ctx, companyID, calendarID); err != nil {
		return err
	}
	return s.repo.DeleteShare(ctx, shareID, calendarID)
}

// ── Публичный доступ по коду (без авторизации) ──

// resolveShare — календарь по коду публичной ссылки (без проверки компании: код —
// capability). Возвращает доменную 404 для неизвестного/отозванного кода.
func (s *Service) resolveShare(ctx context.Context, code string) (*domain.Calendar, error) {
	share, err := s.repo.GetShareByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if share == nil {
		return nil, errShareNotFound
	}
	cal, err := s.repo.GetCalendar(ctx, share.CalendarID)
	if err != nil {
		return nil, err
	}
	if cal == nil {
		return nil, errShareNotFound
	}
	return cal, nil
}

// SharedCalendar — календарь с полями для рендера публичной страницы.
func (s *Service) SharedCalendar(ctx context.Context, code string) (*domain.Calendar, error) {
	cal, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, err
	}
	fields, err := s.repo.ListFields(ctx, cal.ID)
	if err != nil {
		return nil, err
	}
	cal.Fields = fields
	return cal, nil
}

func (s *Service) SharedEntries(ctx context.Context, code string, p EntryListParams) (*EntryList, error) {
	cal, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.listEntriesByCalendar(ctx, cal.ID, p)
}

func (s *Service) SharedExport(ctx context.Context, code string, fieldIDs []int64, p EntryListParams, ids []int64) ([]byte, string, error) {
	cal, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, "", err
	}
	return s.buildExport(ctx, cal, fieldIDs, p, ids)
}

func newShareCode() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
