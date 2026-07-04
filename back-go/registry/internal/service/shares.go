package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

var errShareNotFound = domain.NewError("NOT_FOUND", "Ссылка не найдена или отозвана", 404)

// ── Управление ссылками (требует прав участника компании) ──

func (s *Service) ListShares(ctx context.Context, companyID, registryID int64) ([]*domain.Share, error) {
	if _, err := s.requireRegistry(ctx, companyID, registryID); err != nil {
		return nil, err
	}
	return s.repo.ListShares(ctx, registryID)
}

func (s *Service) CreateShare(ctx context.Context, companyID, registryID, userID int64) (*domain.Share, error) {
	if _, err := s.requireRegistry(ctx, companyID, registryID); err != nil {
		return nil, err
	}
	code, err := records.NewShareCode()
	if err != nil {
		return nil, err
	}
	share := &domain.Share{RegistryID: registryID, Code: code, CreatedBy: &userID}
	if err := s.repo.CreateShare(ctx, share); err != nil {
		return nil, err
	}
	return share, nil
}

func (s *Service) RevokeShare(ctx context.Context, companyID, registryID, shareID int64) error {
	if _, err := s.requireRegistry(ctx, companyID, registryID); err != nil {
		return err
	}
	return s.repo.DeleteShare(ctx, shareID, registryID)
}

// ── Публичный доступ по коду (без авторизации) ──

// resolveShare — реестр по коду публичной ссылки (без проверки компании: код —
// capability). Возвращает доменную 404 для неизвестного/отозванного кода.
func (s *Service) resolveShare(ctx context.Context, code string) (*domain.Registry, error) {
	share, err := s.repo.GetShareByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if share == nil {
		return nil, errShareNotFound
	}
	reg, err := s.repo.GetRegistry(ctx, share.RegistryID)
	if err != nil {
		return nil, err
	}
	if reg == nil {
		return nil, errShareNotFound
	}
	return reg, nil
}

// SharedRegistry — реестр с полями для рендера публичной страницы.
func (s *Service) SharedRegistry(ctx context.Context, code string) (*domain.Registry, error) {
	reg, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, err
	}
	fields, err := s.repo.ListFields(ctx, reg.ID)
	if err != nil {
		return nil, err
	}
	reg.Fields = fields
	return reg, nil
}

func (s *Service) SharedRecords(ctx context.Context, code string, p RecordListParams) (*RecordList, error) {
	reg, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.listRecordsByRegistry(ctx, reg.ID, p)
}

func (s *Service) SharedExport(ctx context.Context, code string, fieldIDs []int64, search string, ids []int64) ([]byte, string, error) {
	reg, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, "", err
	}
	return s.buildExport(ctx, reg, fieldIDs, search, ids)
}
