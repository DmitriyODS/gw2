package service

import (
	"context"
	"encoding/json"

	"github.com/DmitriyODS/gw2/back-go/board/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

// ── Публичные ссылки (владелец) ──────────────────────────────────────

func (s *Service) ListShares(ctx context.Context, userID, boardID int64) ([]*domain.Share, error) {
	if _, err := s.requireOwned(ctx, userID, boardID); err != nil {
		return nil, err
	}
	return s.repo.ListShares(ctx, boardID)
}

func (s *Service) CreateShare(ctx context.Context, userID, boardID int64, access string) (*domain.Share, error) {
	if access != domain.AccessView && access != domain.AccessEdit {
		return nil, domain.ErrBadAccess
	}
	if _, err := s.requireOwned(ctx, userID, boardID); err != nil {
		return nil, err
	}
	code, err := records.NewShareCode()
	if err != nil {
		return nil, err
	}
	share := &domain.Share{BoardID: boardID, Code: code, Access: access}
	if err := s.repo.CreateShare(ctx, share); err != nil {
		return nil, err
	}
	return share, nil
}

func (s *Service) RevokeShare(ctx context.Context, userID, boardID, shareID int64) error {
	if _, err := s.requireOwned(ctx, userID, boardID); err != nil {
		return err
	}
	return s.repo.DeleteShare(ctx, shareID, boardID)
}

// MyCompanies — все компании, в которых состоит пользователь (для выбора
// аудитории шаринга). В отличие от /companies/mine (только админ/создатель) —
// любое членство, ведь делиться можно с любой своей компанией.
func (s *Service) MyCompanies(ctx context.Context, userID int64) ([]*domain.Company, error) {
	return s.users.UserCompanies(ctx, userID)
}

// ── Адресный шаринг досок (пользователь и компания) ────────────────

func (s *Service) ListBoardMembers(ctx context.Context, userID, boardID int64) ([]*domain.Member, error) {
	if _, err := s.requireOwned(ctx, userID, boardID); err != nil {
		return nil, err
	}
	return s.repo.ListBoardMembers(ctx, boardID)
}

// ShareBoard — открыть доску пользователю или компании (идемпотентный upsert:
// меняет право). После шаринга адресаты получают board_member:added — доска
// появляется у них в «Поделились» без перезагрузки.
func (s *Service) ShareBoard(ctx context.Context, userID, boardID int64, target string, targetID int64, canEdit bool) (*domain.Member, error) {
	n, err := s.requireOwned(ctx, userID, boardID)
	if err != nil {
		return nil, err
	}
	member, err := s.applyShare(ctx, userID, target, targetID, canEdit, shareOps{
		upsertUser: func(uid int64, ce bool) error { return s.repo.UpsertBoardUserShare(ctx, boardID, uid, ce) },
		upsertCompany: func(cid int64, name string, ce bool) error {
			return s.repo.UpsertBoardCompanyShare(ctx, boardID, cid, name, ce, userID)
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
	s.bus.Publish(ctx, "board_member:added", s.boardRooms(ctx, boardID, userID), map[string]any{
		"board": boardPayload(n), "can_edit": canEdit,
	})
	return member, nil
}

// UnshareBoard — закрыть доступ пользователя/компании к доске.
func (s *Service) UnshareBoard(ctx context.Context, userID, boardID int64, target string, targetID int64) error {
	if _, err := s.requireOwned(ctx, userID, boardID); err != nil {
		return err
	}
	rooms := s.boardRooms(ctx, boardID, userID) // до удаления — аудитория ещё цела
	switch target {
	case domain.TargetUser:
		if err := s.repo.DeleteBoardUserShare(ctx, boardID, targetID); err != nil {
			return err
		}
	case domain.TargetCompany:
		if err := s.repo.DeleteBoardCompanyShare(ctx, boardID, targetID); err != nil {
			return err
		}
	default:
		return domain.ErrBadTarget
	}
	s.bus.Publish(ctx, "board_member:removed", rooms, map[string]any{"board_id": boardID})
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
	s.bus.Publish(ctx, "board_folder:shared", s.folderRooms(ctx, folderID, userID), map[string]any{
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
	s.bus.Publish(ctx, "board_folder:unshared", rooms, map[string]any{"folder_id": folderID})
	return nil
}

// shareOps — операции upsert конкретного вида объекта (доска/папка).
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

// SharedBoard — доска + режим доступа по коду публичной ссылки.
type SharedBoard struct {
	Board  *domain.Board `json:"board"`
	Access string        `json:"access"`
}

func (s *Service) resolveShare(ctx context.Context, code string) (*domain.Share, *domain.Board, error) {
	share, err := s.repo.GetShareByCode(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	if share == nil {
		return nil, nil, domain.ErrShareNotFound
	}
	n, err := s.repo.GetBoard(ctx, share.BoardID)
	if err != nil {
		return nil, nil, err
	}
	if n == nil {
		return nil, nil, domain.ErrShareNotFound
	}
	return share, n, nil
}

func (s *Service) GetSharedBoard(ctx context.Context, code string) (*SharedBoard, error) {
	share, n, err := s.resolveShare(ctx, code)
	if err != nil {
		return nil, err
	}
	return &SharedBoard{Board: n, Access: share.Access}, nil
}

// UpdateSharedBoard — анонимная правка по edit-ссылке: view-ссылка — 403, поток
// правок по коду троттлится (защита от вандализма).
func (s *Service) UpdateSharedBoard(ctx context.Context, code string, title *string, doc json.RawMessage) (*domain.Board, error) {
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
