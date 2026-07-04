package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
)

// ListMembers — пользователи, которым ежедневник открыт адресно (владелец).
func (s *Service) ListMembers(ctx context.Context, userID, diaryID int64) ([]*domain.Member, error) {
	if _, err := s.requireOwned(ctx, userID, diaryID); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, diaryID)
}

// AddMember — открыть ежедневник пользователю. canCheck — разрешить отмечать
// записи выполненными (иначе только чтение). Идемпотентный upsert: повторный
// вызов обновляет право. Шлёт адресату событие diary:shared — у него
// ежедневник появляется во вкладке «Поделились» без перезагрузки.
func (s *Service) AddMember(ctx context.Context, userID, diaryID, memberID int64, canCheck bool) (*domain.Member, error) {
	d, err := s.requireOwned(ctx, userID, diaryID)
	if err != nil {
		return nil, err
	}
	if memberID == userID {
		return nil, domain.ErrSelfShare
	}
	member, err := s.users.GetUser(ctx, memberID)
	if err != nil {
		return nil, err
	}
	if member == nil || !member.IsActive {
		return nil, domain.ErrMemberNotFound
	}
	if err := s.repo.AddMember(ctx, diaryID, memberID, canCheck); err != nil {
		return nil, err
	}
	if owner, err := s.users.GetUser(ctx, userID); err == nil && owner != nil {
		d.OwnerName, d.OwnerAvatar = owner.FIO, owner.AvatarPath
	}
	d.Shared = true
	d.CanCheck = canCheck
	s.bus.Publish(ctx, "diary:shared", []string{userRoom(memberID)}, diaryPayload(d))
	return &domain.Member{UserID: member.ID, FIO: member.FIO, AvatarPath: member.AvatarPath, CanCheck: canCheck}, nil
}

// RemoveMember — закрыть адресный доступ. Шлёт адресату diary:unshared.
func (s *Service) RemoveMember(ctx context.Context, userID, diaryID, memberID int64) error {
	if _, err := s.requireOwned(ctx, userID, diaryID); err != nil {
		return err
	}
	if err := s.repo.RemoveMember(ctx, diaryID, memberID); err != nil {
		return err
	}
	s.bus.Publish(ctx, "diary:unshared", []string{userRoom(memberID)}, map[string]any{
		"id": diaryID, "owner_id": userID,
	})
	return nil
}
