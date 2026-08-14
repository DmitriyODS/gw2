package service

import (
	"context"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

var (
	errShareNotFound = domain.NewError("NOT_FOUND", "Ссылка не найдена или отозвана", 404)
	errShareLevel    = domain.NewError("SHARE_READ_ONLY", "Эта ссылка не даёт такого доступа", 403)
	// Ссылка «только для своих»: пока гость не вошёл, реестра он не увидит.
	errShareAuth = domain.NewError("SHARE_AUTH_REQUIRED",
		"Чтобы открыть реестр по этой ссылке, войдите в аккаунт", 401)
)

/* Поделиться реестром можно тремя путями, и уровень доступа у каждого свой:

     ссылка      — код-capability в адресе (плюс необязательный вход в аккаунт);
     пользователь — адресно коллеге;
     компания    — сразу всем её участникам.

   Раздавать доступ вправе владелец и уровень admin: это управление самим
   реестром, а не работа с записями. */

// ── Адресный доступ (люди и компании) ──

func (s *Service) ListUserShares(ctx context.Context, userID, registryID int64) ([]*domain.UserShare, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.require(ctx, a, registryID, domain.AccessAdmin); err != nil {
		return nil, err
	}
	return s.repo.ListUserShares(ctx, registryID)
}

// ShareTarget — кому выдаём доступ и какой.
type ShareTarget struct {
	UserID    *int64
	CompanyID *int64
	Access    string
}

// ShareWith — выдать (или изменить) адресный доступ. Компания берётся только из
// тех, где состоит выдающий: раздать реестр чужой компании нельзя.
func (s *Service) ShareWith(ctx context.Context, userID, registryID int64, targets []ShareTarget) ([]*domain.UserShare, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	reg, err := s.require(ctx, a, registryID, domain.AccessAdmin)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, domain.ErrNoAudience
	}
	own := map[int64]bool{}
	for _, id := range a.Companies {
		own[id] = true
	}
	for _, t := range targets {
		switch {
		case t.UserID != nil:
			// Владельцу выдавать нечего: у него и так всё.
			if *t.UserID == reg.OwnerID {
				return nil, domain.ErrShareSelf
			}
		case t.CompanyID != nil:
			if !own[*t.CompanyID] {
				return nil, domain.ErrForbidden
			}
		default:
			return nil, domain.ErrNoAudience
		}
		share := &domain.UserShare{
			RegistryID: registryID, UserID: t.UserID, CompanyID: t.CompanyID,
			Access: domain.NormalizeShareAccess(t.Access), CreatedBy: &a.UserID,
		}
		if err := s.repo.PutUserShare(ctx, share); err != nil {
			return nil, err
		}
	}
	// Аудитория изменилась — новые адресаты должны увидеть реестр сразу.
	s.publish(ctx, registryID, "registry:shared", map[string]any{"id": registryID})
	return s.repo.ListUserShares(ctx, registryID)
}

// Unshare — отозвать адресный доступ.
func (s *Service) Unshare(ctx context.Context, userID, registryID int64, targetUser, targetCompany *int64) error {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return err
	}
	if _, err := s.require(ctx, a, registryID, domain.AccessAdmin); err != nil {
		return err
	}
	// Комнаты берём ДО отзыва: тому, у кого доступ забрали, событие тоже нужно —
	// иначе реестр останется висеть в его списке до перезагрузки.
	rooms := s.audience(ctx, registryID)
	if err := s.repo.DeleteUserShare(ctx, registryID, targetUser, targetCompany); err != nil {
		return err
	}
	s.bus.Publish(ctx, "registry:shared", rooms, map[string]any{"id": registryID})
	return nil
}

// Directory — кандидаты в адресаты: коллеги из компаний выдающего.
func (s *Service) Directory(ctx context.Context, userID int64, query string, limit int) ([]*domain.User, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.users.SearchDirectory(ctx, a.Companies, strings.TrimSpace(query), limit)
}

// Companies — компании, которым выдающий может отдать реестр (те, где состоит).
func (s *Service) Companies(ctx context.Context, userID int64) ([]map[string]any, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(a.Companies))
	for _, id := range a.Companies {
		name, err := s.users.CompanyName(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "name": name})
	}
	return out, nil
}

// ── Внешние ссылки ──

func (s *Service) ListShares(ctx context.Context, userID, registryID int64) ([]*domain.Share, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.require(ctx, a, registryID, domain.AccessAdmin); err != nil {
		return nil, err
	}
	return s.repo.ListShares(ctx, registryID)
}

// ShareParams — настройки внешней ссылки.
type ShareParams struct {
	Name        string
	Access      string
	RequireAuth bool
}

func (s *Service) CreateShare(ctx context.Context, userID, registryID int64, p ShareParams) (*domain.Share, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.require(ctx, a, registryID, domain.AccessAdmin); err != nil {
		return nil, err
	}
	code, err := records.NewShareCode()
	if err != nil {
		return nil, err
	}
	share := &domain.Share{
		RegistryID:  registryID,
		Code:        code,
		Name:        strings.TrimSpace(p.Name),
		Access:      domain.NormalizeShareAccess(p.Access),
		RequireAuth: p.RequireAuth,
		CreatedBy:   &userID,
	}
	if err := s.repo.CreateShare(ctx, share); err != nil {
		return nil, err
	}
	return share, nil
}

func (s *Service) UpdateShare(ctx context.Context, userID, registryID, shareID int64, p ShareParams) error {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return err
	}
	if _, err := s.require(ctx, a, registryID, domain.AccessAdmin); err != nil {
		return err
	}
	return s.repo.UpdateShare(ctx, shareID, registryID,
		strings.TrimSpace(p.Name), domain.NormalizeShareAccess(p.Access), p.RequireAuth)
}

func (s *Service) RevokeShare(ctx context.Context, userID, registryID, shareID int64) error {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return err
	}
	if _, err := s.require(ctx, a, registryID, domain.AccessAdmin); err != nil {
		return err
	}
	return s.repo.DeleteShare(ctx, shareID, registryID)
}

// ShareVisits — журнал переходов по ссылке.
func (s *Service) ShareVisits(ctx context.Context, userID, registryID, shareID int64, limit int) ([]*domain.ShareVisit, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.require(ctx, a, registryID, domain.AccessAdmin); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	return s.repo.ListVisits(ctx, shareID, limit)
}

// ── Публичный доступ по коду ──

// SharedView — реестр по внешней ссылке вместе с уровнем доступа ЭТОЙ ссылки:
// по нему публичная страница решает, что показывать.
type SharedView struct {
	*domain.Registry
	Access string `json:"access"`
}

// Visitor — кто открывает ссылку. У гостя нет аккаунта, поэтому в журнале
// остаётся только адрес и браузер.
type Visitor struct {
	UserID    *int64
	IP        string
	UserAgent string
}

// resolveShare — реестр и сама ссылка по коду. Проверяет требование входа:
// ссылка с require_auth не отдаёт ничего, пока гость не представился.
func (s *Service) resolveShare(ctx context.Context, code string, v Visitor) (*domain.Registry, *domain.Share, error) {
	share, err := s.repo.GetShareByCode(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	if share == nil {
		return nil, nil, errShareNotFound
	}
	if share.RequireAuth && (v.UserID == nil || *v.UserID == 0) {
		return nil, nil, errShareAuth
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

// resolveShareAt — то же с требованием уровня: ссылка на просмотр не даёт
// править, ссылка на правку — менять структуру.
func (s *Service) resolveShareAt(ctx context.Context, code string, v Visitor, want string) (*domain.Registry, error) {
	reg, share, err := s.resolveShare(ctx, code, v)
	if err != nil {
		return nil, err
	}
	if !domain.AccessAtLeast(share.Access, want) {
		return nil, errShareLevel
	}
	return reg, nil
}

// SharedRegistry — реестр с полями для рендера публичной страницы. Здесь же
// пишется журнал посещений: открытие страницы и есть переход по ссылке.
func (s *Service) SharedRegistry(ctx context.Context, code string, v Visitor) (*SharedView, error) {
	reg, share, err := s.resolveShare(ctx, code, v)
	if err != nil {
		return nil, err
	}
	fields, err := s.repo.ListFields(ctx, reg.ID)
	if err != nil {
		return nil, err
	}
	reg.Fields = fields
	// Журнал ведём best-effort: не записавшийся переход не повод не пустить
	// человека к реестру.
	if err := s.repo.LogVisit(ctx, &domain.ShareVisit{
		ShareID: share.ID, UserID: v.UserID, IP: v.IP, UserAgent: v.UserAgent,
	}); err != nil {
		s.log.Warn("registry.visit_log_failed", "share_id", share.ID, "error", err)
	}
	return &SharedView{Registry: reg, Access: share.Access}, nil
}

func (s *Service) SharedRecords(ctx context.Context, code string, v Visitor, p RecordListParams) (*RecordList, error) {
	reg, _, err := s.resolveShare(ctx, code, v)
	if err != nil {
		return nil, err
	}
	return s.listRecordsByRegistry(ctx, reg, p)
}

func (s *Service) SharedExport(ctx context.Context, code string, v Visitor, p ExportParams) ([]byte, string, error) {
	reg, _, err := s.resolveShare(ctx, code, v)
	if err != nil {
		return nil, "", err
	}
	return s.buildExport(ctx, reg, p)
}

// ── Правка по ссылке уровня edit ──
//
// Автора у таких записей нет, если гость не вошёл (created_by остаётся пустым);
// в остальном путь тот же, что у участника, — те же проверки значений и те же
// события той же аудитории.

func (s *Service) SharedCreateRecord(ctx context.Context, code string, v Visitor, data map[string]any) (*domain.Record, error) {
	reg, err := s.resolveShareAt(ctx, code, v, domain.AccessEdit)
	if err != nil {
		return nil, err
	}
	return s.createRecordIn(ctx, reg, v.UserID, data)
}

func (s *Service) SharedUpdateRecord(ctx context.Context, code string, v Visitor, recordID int64, data map[string]any) (*domain.Record, error) {
	reg, err := s.resolveShareAt(ctx, code, v, domain.AccessEdit)
	if err != nil {
		return nil, err
	}
	return s.updateRecordIn(ctx, reg, recordID, data)
}

func (s *Service) SharedDeleteRecord(ctx context.Context, code string, v Visitor, recordID int64) error {
	reg, err := s.resolveShareAt(ctx, code, v, domain.AccessEdit)
	if err != nil {
		return err
	}
	rec, err := s.recordIn(ctx, reg, recordID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteRecord(ctx, recordID); err != nil {
		return err
	}
	s.removeRecordFiles(ctx, reg, rec)
	s.publish(ctx, reg.ID, "record:deleted", map[string]any{
		"id": recordID, "registry_id": reg.ID,
	})
	return nil
}

/* ── Правка структуры по ссылке уровня admin ──
   «Администрирование» тем и отличается от «редактирования», что позволяет
   менять сам реестр, а не только его записи. Проверка уровня — та же, что и у
   остальных операций по коду; дальше идёт общее ядро, что и для участника. */

func (s *Service) SharedUpdateRegistry(ctx context.Context, code string, v Visitor, p RegistryPatch) (*domain.Registry, error) {
	reg, err := s.resolveShareAt(ctx, code, v, domain.AccessAdmin)
	if err != nil {
		return nil, err
	}
	return s.updateRegistryIn(ctx, reg, p)
}

func (s *Service) SharedReplaceFields(ctx context.Context, code string, v Visitor, fields []domain.Field) (*domain.Registry, error) {
	reg, err := s.resolveShareAt(ctx, code, v, domain.AccessAdmin)
	if err != nil {
		return nil, err
	}
	return s.replaceFieldsIn(ctx, reg, fields)
}

// SharedUpload — файл/картинка поля записи, загруженная по ссылке уровня edit.
// Место тратится из квоты владельца реестра: своего аккаунта у гостя может не
// быть, а файл принадлежит именно реестру.
func (s *Service) SharedUpload(ctx context.Context, code string, v Visitor, fileName, mime string, data []byte) (*domain.UploadedFile, error) {
	reg, err := s.resolveShareAt(ctx, code, v, domain.AccessEdit)
	if err != nil {
		return nil, err
	}
	return s.saveUpload(ctx, reg, fileName, mime, data)
}
