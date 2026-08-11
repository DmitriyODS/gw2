package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

var (
	errShareNotFound = domain.NewError("NOT_FOUND", "Ссылка не найдена или отозвана", 404)
	errShareReadOnly = domain.NewError("SHARE_READ_ONLY", "Эта ссылка открыта только для просмотра", 403)
)

// ── Управление ссылками (требует прав участника компании) ──

func (s *Service) ListShares(ctx context.Context, companyID, registryID int64) ([]*domain.Share, error) {
	if _, err := s.requireRegistry(ctx, companyID, registryID); err != nil {
		return nil, err
	}
	return s.repo.ListShares(ctx, registryID)
}

func (s *Service) CreateShare(ctx context.Context, companyID, registryID, userID int64, access string) (*domain.Share, error) {
	if _, err := s.requireRegistry(ctx, companyID, registryID); err != nil {
		return nil, err
	}
	code, err := records.NewShareCode()
	if err != nil {
		return nil, err
	}
	share := &domain.Share{
		RegistryID: registryID,
		Code:       code,
		Access:     domain.NormalizeShareAccess(access),
		CreatedBy:  &userID,
	}
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

// SharedView — реестр по внешней ссылке вместе с уровнем доступа ЭТОЙ ссылки:
// по нему публичная страница решает, показывать ли правку.
type SharedView struct {
	*domain.Registry
	Access string `json:"access"`
}

// resolveShare — реестр и сама ссылка по коду (без проверки компании: код —
// capability). Возвращает доменную 404 для неизвестного/отозванного кода.
func (s *Service) resolveShare(ctx context.Context, code string) (*domain.Registry, *domain.Share, error) {
	share, err := s.repo.GetShareByCode(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	if share == nil {
		return nil, nil, errShareNotFound
	}
	reg, err := s.repo.GetRegistry(ctx, share.RegistryID)
	if err != nil {
		return nil, nil, err
	}
	if reg == nil {
		return nil, nil, errShareNotFound
	}
	return reg, share, nil
}

// resolveShareEdit — то же для пишущих операций: ссылка на просмотр их не даёт.
func (s *Service) resolveShareEdit(ctx context.Context, code string) (*domain.Registry, error) {
	reg, share, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, err
	}
	if share.Access != domain.ShareEdit {
		return nil, errShareReadOnly
	}
	return reg, nil
}

// SharedRegistry — реестр с полями для рендера публичной страницы.
func (s *Service) SharedRegistry(ctx context.Context, code string) (*SharedView, error) {
	reg, share, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, err
	}
	fields, err := s.repo.ListFields(ctx, reg.ID)
	if err != nil {
		return nil, err
	}
	reg.Fields = fields
	return &SharedView{Registry: reg, Access: share.Access}, nil
}

func (s *Service) SharedRecords(ctx context.Context, code string, p RecordListParams) (*RecordList, error) {
	reg, _, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.listRecordsByRegistry(ctx, reg.ID, p)
}

func (s *Service) SharedExport(ctx context.Context, code string, fieldIDs []int64, search string, ids []int64) ([]byte, string, error) {
	reg, _, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, "", err
	}
	return s.buildExport(ctx, reg, fieldIDs, search, ids)
}

// ── Правка по ссылке уровня edit ──
//
// Автора у таких записей нет (гость не представляется), поэтому created_by
// остаётся пустым; в остальном путь тот же, что у участника компании, — те же
// проверки значений и те же события в комнату компании.

func (s *Service) SharedCreateRecord(ctx context.Context, code string, data map[string]any) (*domain.Record, error) {
	reg, err := s.resolveShareEdit(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.createRecordIn(ctx, reg, nil, data)
}

func (s *Service) SharedUpdateRecord(ctx context.Context, code string, recordID int64, data map[string]any) (*domain.Record, error) {
	reg, err := s.resolveShareEdit(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.updateRecordIn(ctx, reg, recordID, data)
}

// SharedUpload — файл/картинка поля записи, загруженная по ссылке уровня edit.
// Место тратится из квоты компании-владельца реестра: своего аккаунта у гостя
// нет, а файл принадлежит именно ей.
func (s *Service) SharedUpload(ctx context.Context, code, fileName, mime string, data []byte) (*domain.UploadedFile, error) {
	reg, err := s.resolveShareEdit(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.SaveUpload(ctx, reg.CompanyID, 0, fileName, mime, data)
}
