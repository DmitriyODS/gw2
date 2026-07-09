package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// ListMembers — адресаты заметки (только владелец).
func (s *Service) ListMembers(ctx context.Context, userID, noteID int64) ([]*domain.NoteMember, error) {
	if _, err := s.requireOwned(ctx, userID, noteID); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, noteID)
}

// AddMember — открыть заметку пользователю платформы. canEdit — разрешить
// правку title/doc (иначе только чтение). Идемпотентный upsert: повторный
// вызов меняет право. Адресату шлётся note_member:added — заметка появляется
// у него во вкладке «Поделились» без перезагрузки.
func (s *Service) AddMember(ctx context.Context, userID, noteID, memberID int64, canEdit bool) (*domain.NoteMember, error) {
	n, err := s.requireOwned(ctx, userID, noteID)
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
	if err := s.repo.UpsertMember(ctx, noteID, memberID, canEdit); err != nil {
		return nil, err
	}
	if owner, err := s.users.GetUser(ctx, userID); err == nil && owner != nil {
		n.OwnerName, n.OwnerAvatar = owner.FIO, owner.AvatarPath
	}
	s.bus.Publish(ctx, "note_member:added", []string{userRoom(memberID)}, map[string]any{
		"note": notePayload(n), "can_edit": canEdit,
	})
	return &domain.NoteMember{UserID: member.ID, FIO: member.FIO, AvatarPath: member.AvatarPath, CanEdit: canEdit}, nil
}

// RemoveMember — закрыть адресный доступ (только владелец). Адресату шлётся
// note_member:removed — заметка пропадает из «Поделились» вживую.
func (s *Service) RemoveMember(ctx context.Context, userID, noteID, memberID int64) error {
	if _, err := s.requireOwned(ctx, userID, noteID); err != nil {
		return err
	}
	if err := s.repo.DeleteMember(ctx, noteID, memberID); err != nil {
		return err
	}
	s.bus.Publish(ctx, "note_member:removed", []string{userRoom(memberID)}, map[string]any{"note_id": noteID})
	return nil
}
