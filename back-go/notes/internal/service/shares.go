package service

import (
	"context"
	"encoding/json"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

// ── Управление публичными ссылками (владелец) ──

func (s *Service) ListShares(ctx context.Context, userID, noteID int64) ([]*domain.Share, error) {
	if _, err := s.requireOwned(ctx, userID, noteID); err != nil {
		return nil, err
	}
	return s.repo.ListShares(ctx, noteID)
}

func (s *Service) CreateShare(ctx context.Context, userID, noteID int64, access string) (*domain.Share, error) {
	if access != domain.AccessView && access != domain.AccessEdit {
		return nil, domain.ErrBadAccess
	}
	if _, err := s.requireOwned(ctx, userID, noteID); err != nil {
		return nil, err
	}
	code, err := records.NewShareCode()
	if err != nil {
		return nil, err
	}
	share := &domain.Share{NoteID: noteID, Code: code, Access: access}
	if err := s.repo.CreateShare(ctx, share); err != nil {
		return nil, err
	}
	return share, nil
}

func (s *Service) RevokeShare(ctx context.Context, userID, noteID, shareID int64) error {
	if _, err := s.requireOwned(ctx, userID, noteID); err != nil {
		return err
	}
	return s.repo.DeleteShare(ctx, shareID, noteID)
}

// ── Публичный доступ по коду (без авторизации) ──

// SharedNote — заметка + режим доступа по коду публичной ссылки.
type SharedNote struct {
	Note   *domain.Note `json:"note"`
	Access string       `json:"access"`
}

// resolveShare — ссылка и заметка по коду (код — capability).
func (s *Service) resolveShare(ctx context.Context, code string) (*domain.Share, *domain.Note, error) {
	share, err := s.repo.GetShareByCode(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	if share == nil {
		return nil, nil, domain.ErrShareNotFound
	}
	n, err := s.repo.GetNote(ctx, share.NoteID)
	if err != nil {
		return nil, nil, err
	}
	if n == nil {
		return nil, nil, domain.ErrShareNotFound
	}
	return share, n, nil
}

func (s *Service) GetSharedNote(ctx context.Context, code string) (*SharedNote, error) {
	share, n, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, err
	}
	return &SharedNote{Note: n, Access: share.Access}, nil
}

// UpdateSharedNote — анонимная правка по edit-ссылке: view-ссылка — 403,
// поток правок по коду троттлится (защита от вандализма). Владелец получает
// note:updated в свою комнату как обычно.
func (s *Service) UpdateSharedNote(ctx context.Context, code string, title *string, doc json.RawMessage) (*domain.Note, error) {
	share, n, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, err
	}
	if share.Access != domain.AccessEdit {
		return nil, domain.ErrReadOnly
	}
	if !s.limiter.Allow(ctx, code) {
		return nil, domain.ErrRateLimited
	}
	return s.applyUpdate(ctx, n, title, doc)
}
