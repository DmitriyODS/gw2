package service

import (
	"context"
	"encoding/json"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

// ── Публичные ссылки (владелец) ──────────────────────────────────────

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

// MyCompanies — все компании, в которых состоит пользователь (для выбора
// аудитории шаринга). В отличие от /companies/mine (только админ/создатель) —
// любое членство, ведь делиться можно с любой своей компанией.
func (s *Service) MyCompanies(ctx context.Context, userID int64) ([]*domain.Company, error) {
	return s.users.UserCompanies(ctx, userID)
}

// ── Адресный шаринг заметок (пользователь и компания) ────────────────

func (s *Service) ListNoteMembers(ctx context.Context, userID, noteID int64) ([]*domain.Member, error) {
	if _, err := s.requireOwned(ctx, userID, noteID); err != nil {
		return nil, err
	}
	return s.repo.ListNoteMembers(ctx, noteID)
}

// ShareNote — открыть заметку пользователю или компании (идемпотентный upsert:
// меняет право). После шаринга адресаты получают note_member:added — заметка
// появляется у них в «Поделились» без перезагрузки.
func (s *Service) ShareNote(ctx context.Context, userID, noteID int64, target string, targetID int64, canEdit bool) (*domain.Member, error) {
	n, err := s.requireOwned(ctx, userID, noteID)
	if err != nil {
		return nil, err
	}
	member, err := s.applyShare(ctx, userID, target, targetID, canEdit, shareOps{
		upsertUser: func(uid int64, ce bool) error { return s.repo.UpsertNoteUserShare(ctx, noteID, uid, ce) },
		upsertCompany: func(cid int64, name string, ce bool) error {
			return s.repo.UpsertNoteCompanyShare(ctx, noteID, cid, name, ce, userID)
		},
	})
	if err != nil {
		return nil, err
	}
	// Владелец — для плитки у адресатов; событие несёт полный тайл + can_edit.
	if owner, e := s.users.GetUser(ctx, userID); e == nil && owner != nil {
		n.OwnerName, n.OwnerAvatar = owner.FIO, owner.AvatarPath
	}
	n.MyAccess = domain.AccessView
	if canEdit {
		n.MyAccess = domain.AccessEdit
	}
	s.bus.Publish(ctx, "note_member:added", s.noteRooms(ctx, noteID, userID), map[string]any{
		"note": notePayload(n), "can_edit": canEdit,
	})
	return member, nil
}

// UnshareNote — закрыть доступ пользователя/компании к заметке.
func (s *Service) UnshareNote(ctx context.Context, userID, noteID int64, target string, targetID int64) error {
	if _, err := s.requireOwned(ctx, userID, noteID); err != nil {
		return err
	}
	rooms := s.noteRooms(ctx, noteID, userID) // до удаления — аудитория ещё цела
	switch target {
	case domain.TargetUser:
		if err := s.repo.DeleteNoteUserShare(ctx, noteID, targetID); err != nil {
			return err
		}
	case domain.TargetCompany:
		if err := s.repo.DeleteNoteCompanyShare(ctx, noteID, targetID); err != nil {
			return err
		}
	default:
		return domain.ErrBadTarget
	}
	s.bus.Publish(ctx, "note_member:removed", rooms, map[string]any{"note_id": noteID})
	return nil
}

// ── Адресный шаринг папок (пользователь и компания) ──────────────────

func (s *Service) ListFolderMembers(ctx context.Context, userID, folderID int64) ([]*domain.Member, error) {
	if _, err := s.requireFolderOwned(ctx, userID, folderID); err != nil {
		return nil, err
	}
	return s.repo.ListFolderMembers(ctx, folderID)
}

// ShareFolder — открыть папку (со всем поддеревом) пользователю или компании.
func (s *Service) ShareFolder(ctx context.Context, userID, folderID int64, target string, targetID int64, canEdit bool) (*domain.Member, error) {
	f, err := s.requireFolderOwned(ctx, userID, folderID)
	if err != nil {
		return nil, err
	}
	member, err := s.applyShare(ctx, userID, target, targetID, canEdit, shareOps{
		upsertUser: func(uid int64, ce bool) error { return s.repo.UpsertFolderUserShare(ctx, folderID, uid, ce) },
		upsertCompany: func(cid int64, name string, ce bool) error {
			return s.repo.UpsertFolderCompanyShare(ctx, folderID, cid, name, ce, userID)
		},
	})
	if err != nil {
		return nil, err
	}
	if owner, e := s.users.GetUser(ctx, userID); e == nil && owner != nil {
		f.OwnerName, f.OwnerAvatar = owner.FIO, owner.AvatarPath
	}
	f.MyAccess = domain.AccessView
	if canEdit {
		f.MyAccess = domain.AccessEdit
	}
	f.SharedByMe = true
	s.bus.Publish(ctx, "note_folder:shared", s.folderRooms(ctx, folderID, userID), map[string]any{
		"folder": folderPayload(f), "can_edit": canEdit,
	})
	return member, nil
}

// UnshareFolder — закрыть доступ пользователя/компании к папке.
func (s *Service) UnshareFolder(ctx context.Context, userID, folderID int64, target string, targetID int64) error {
	if _, err := s.requireFolderOwned(ctx, userID, folderID); err != nil {
		return err
	}
	rooms := s.folderRooms(ctx, folderID, userID)
	switch target {
	case domain.TargetUser:
		if err := s.repo.DeleteFolderUserShare(ctx, folderID, targetID); err != nil {
			return err
		}
	case domain.TargetCompany:
		if err := s.repo.DeleteFolderCompanyShare(ctx, folderID, targetID); err != nil {
			return err
		}
	default:
		return domain.ErrBadTarget
	}
	s.bus.Publish(ctx, "note_folder:unshared", rooms, map[string]any{"folder_id": folderID})
	return nil
}

// shareOps — операции upsert конкретного вида объекта (заметка/папка).
type shareOps struct {
	upsertUser    func(userID int64, canEdit bool) error
	upsertCompany func(companyID int64, name string, canEdit bool) error
}

// applyShare — общая проверка аудитории + upsert доступа; возвращает Member для
// ответа. Компанией можно делиться только будучи её членом (name берётся оттуда).
func (s *Service) applyShare(ctx context.Context, ownerID int64, target string, targetID int64, canEdit bool, ops shareOps) (*domain.Member, error) {
	switch target {
	case domain.TargetUser:
		if targetID == ownerID {
			return nil, domain.ErrSelfShare
		}
		u, err := s.users.GetUser(ctx, targetID)
		if err != nil {
			return nil, err
		}
		if u == nil || !u.IsActive {
			return nil, domain.ErrMemberNotFound
		}
		if err := ops.upsertUser(targetID, canEdit); err != nil {
			return nil, err
		}
		return &domain.Member{Target: domain.TargetUser, UserID: u.ID, FIO: u.FIO, AvatarPath: u.AvatarPath, CanEdit: canEdit}, nil
	case domain.TargetCompany:
		ok, name, err := s.users.IsCompanyMember(ctx, ownerID, targetID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, domain.ErrNotCompanyMember
		}
		if err := ops.upsertCompany(targetID, name, canEdit); err != nil {
			return nil, err
		}
		return &domain.Member{Target: domain.TargetCompany, CompanyID: targetID, CompanyName: name, CanEdit: canEdit}, nil
	default:
		return nil, domain.ErrBadTarget
	}
}

// ── Публичный доступ по коду (без авторизации) ──────────────────────

// SharedNote — заметка + режим доступа по коду публичной ссылки.
type SharedNote struct {
	Note   *domain.Note `json:"note"`
	Access string       `json:"access"`
}

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
	n.TagIDs = []int64{}
	return &SharedNote{Note: n, Access: share.Access}, nil
}

// UpdateSharedNote — анонимная правка по edit-ссылке: view-ссылка — 403, поток
// правок по коду троттлится (защита от вандализма).
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
