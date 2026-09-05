package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/schedule/internal/domain"
)

/* Шаринг расписания — только на ЧТЕНИЕ, двумя путями: публичной ссылкой
   (код-capability в URL, вход не нужен) и адресной выдачей человеку либо
   компании. Вести занятия может владелец, поэтому уровней доступа у раздела
   нет вовсе. */

func (s *Service) ListShares(ctx context.Context, userID, scheduleID int64) ([]*domain.Share, error) {
	if _, err := s.requireOwner(ctx, userID, scheduleID); err != nil {
		return nil, err
	}
	return s.repo.ListShares(ctx, scheduleID)
}

func (s *Service) CreateShare(ctx context.Context, userID, scheduleID int64) (*domain.Share, error) {
	sc, err := s.requireOwner(ctx, userID, scheduleID)
	if err != nil {
		return nil, err
	}
	code, err := records.NewShareCode()
	if err != nil {
		return nil, err
	}
	share := &domain.Share{ScheduleID: sc.ID, Code: code, CreatedBy: &userID}
	if err := s.repo.CreateShare(ctx, share); err != nil {
		return nil, err
	}
	return share, nil
}

func (s *Service) RevokeShare(ctx context.Context, userID, scheduleID, shareID int64) error {
	if _, err := s.requireOwner(ctx, userID, scheduleID); err != nil {
		return err
	}
	return s.repo.DeleteShare(ctx, shareID, scheduleID)
}

// SharedView — расписание по коду публичной ссылки: read-only и без сессии.
func (s *Service) SharedView(ctx context.Context, code string) (*View, error) {
	share, err := s.repo.GetShareByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if share == nil {
		return nil, domain.ErrShareNotFound
	}
	sc, err := s.repo.GetSchedule(ctx, share.ScheduleID)
	if err != nil {
		return nil, err
	}
	if sc == nil {
		return nil, domain.ErrShareNotFound
	}
	sc.Shared = true
	return s.view(ctx, sc, false)
}

func (s *Service) ListUserShares(ctx context.Context, userID, scheduleID int64) ([]*domain.UserShare, error) {
	if _, err := s.requireOwner(ctx, userID, scheduleID); err != nil {
		return nil, err
	}
	return s.repo.ListUserShares(ctx, scheduleID)
}

// ShareWith — открыть расписание человеку либо компании (адресат ровно один).
func (s *Service) ShareWith(ctx context.Context, userID, scheduleID int64, targetUser, targetCompany *int64) (*domain.UserShare, error) {
	sc, err := s.requireOwner(ctx, userID, scheduleID)
	if err != nil {
		return nil, err
	}
	if (targetUser == nil) == (targetCompany == nil) {
		return nil, domain.ErrNoAudience
	}
	if targetUser != nil && *targetUser == userID {
		return nil, domain.ErrShareSelf
	}
	share := &domain.UserShare{
		ScheduleID: sc.ID, UserID: targetUser, CompanyID: targetCompany, CreatedBy: &userID,
	}
	if err := s.repo.PutUserShare(ctx, share); err != nil {
		return nil, err
	}
	// Событие уходит уже НОВОЙ аудитории — адресат сразу видит расписание во
	// вкладке «Поделились», без перезагрузки.
	s.publish(ctx, sc.ID, "schedule:shared", schedulePayload(sc))
	return share, nil
}

func (s *Service) Unshare(ctx context.Context, userID, scheduleID int64, targetUser, targetCompany *int64) error {
	sc, err := s.requireOwner(ctx, userID, scheduleID)
	if err != nil {
		return err
	}
	if (targetUser == nil) == (targetCompany == nil) {
		return domain.ErrNoAudience
	}
	// Пока адресат ещё в аудитории — сообщаем ему, что расписание уходит.
	s.publish(ctx, sc.ID, "schedule:unshared", map[string]any{"id": sc.ID})
	return s.repo.DeleteUserShare(ctx, sc.ID, targetUser, targetCompany)
}

// Directory — коллеги, которым можно открыть расписание.
func (s *Service) Directory(ctx context.Context, userID int64, query string, limit int) ([]*domain.UserRef, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.users.Colleagues(ctx, userID, query, limit)
}

// Companies — компании пользователя (им тоже можно открыть расписание).
func (s *Service) Companies(ctx context.Context, userID int64) ([]*domain.CompanyRef, error) {
	return s.users.Companies(ctx, userID)
}
