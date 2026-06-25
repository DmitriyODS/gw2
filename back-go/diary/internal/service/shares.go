package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/DmitriyODS/gw2/back-go/diary/internal/domain"
)

// ── Управление публичными ссылками (владелец) ──

func (s *Service) ListShares(ctx context.Context, userID, diaryID int64) ([]*domain.Share, error) {
	if _, err := s.requireOwned(ctx, userID, diaryID); err != nil {
		return nil, err
	}
	return s.repo.ListShares(ctx, diaryID)
}

func (s *Service) CreateShare(ctx context.Context, userID, diaryID int64) (*domain.Share, error) {
	if _, err := s.requireOwned(ctx, userID, diaryID); err != nil {
		return nil, err
	}
	code, err := newShareCode()
	if err != nil {
		return nil, err
	}
	share := &domain.Share{DiaryID: diaryID, Code: code, CreatedBy: &userID}
	if err := s.repo.CreateShare(ctx, share); err != nil {
		return nil, err
	}
	return share, nil
}

func (s *Service) RevokeShare(ctx context.Context, userID, diaryID, shareID int64) error {
	if _, err := s.requireOwned(ctx, userID, diaryID); err != nil {
		return err
	}
	return s.repo.DeleteShare(ctx, shareID, diaryID)
}

// ── Публичный доступ по коду (без авторизации) ──

// resolveShare — ежедневник по коду публичной ссылки (код — capability).
func (s *Service) resolveShare(ctx context.Context, code string) (*domain.Diary, error) {
	share, err := s.repo.GetShareByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if share == nil {
		return nil, domain.ErrShareNotFound
	}
	d, err := s.repo.GetDiary(ctx, share.DiaryID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, domain.ErrShareNotFound
	}
	return d, nil
}

func (s *Service) SharedDiary(ctx context.Context, code string) (*domain.Diary, error) {
	d, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, err
	}
	d.Shared = true
	return d, nil
}

func (s *Service) SharedEntries(ctx context.Context, code string, p ListParams) (*EntryList, error) {
	d, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.listEntries(ctx, d.ID, p)
}

func (s *Service) SharedExport(ctx context.Context, code string, p ListParams, ids []int64) ([]byte, string, error) {
	d, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, "", err
	}
	return s.buildExport(ctx, d, p, ids)
}

func newShareCode() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
