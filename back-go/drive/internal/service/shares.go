package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

/* Доступ к содержимому диска — два пути, как в заметках и досках:
   публичная ссылка (код в адресе = capability) и адресная выдача человеку или
   компании. Доступ, выданный на папку, действует на всё её поддерево. */

// Target — на что выдаётся доступ: файл или папка.
type Target struct {
	FileID   *int64
	FolderID *int64
}

// CreateShare — публичная ссылка. Выдаёт только владелец: раздавать чужое,
// получив доступ, нельзя.
func (s *Service) CreateShare(ctx context.Context, userID int64, t Target) (*domain.Share, error) {
	if err := s.requireOwnTarget(ctx, userID, t); err != nil {
		return nil, err
	}
	code, err := records.NewShareCode()
	if err != nil {
		return nil, err
	}
	share := &domain.Share{FileID: t.FileID, FolderID: t.FolderID, Code: code, CreatedBy: userID}
	if err := s.repo.CreateShare(ctx, share); err != nil {
		return nil, err
	}
	return share, nil
}

func (s *Service) ListShares(ctx context.Context, userID int64, t Target) ([]*domain.Share, []*domain.UserShare, error) {
	if err := s.requireOwnTarget(ctx, userID, t); err != nil {
		return nil, nil, err
	}
	links, err := s.repo.ListShares(ctx, t.FileID, t.FolderID)
	if err != nil {
		return nil, nil, err
	}
	people, err := s.repo.ListUserShares(ctx, t.FileID, t.FolderID)
	if err != nil {
		return nil, nil, err
	}
	return links, people, nil
}

func (s *Service) DeleteShare(ctx context.Context, userID, id int64) error {
	return s.repo.DeleteShare(ctx, id, userID)
}

// ShareTo — открыть доступ человеку или компании (повторная выдача меняет права).
func (s *Service) ShareTo(ctx context.Context, userID int64, t Target, toUser, toCompany *int64, canEdit bool) (*domain.UserShare, error) {
	if err := s.requireOwnTarget(ctx, userID, t); err != nil {
		return nil, err
	}
	if (toUser == nil) == (toCompany == nil) {
		return nil, domain.ErrValidation // ровно один адресат
	}
	share := &domain.UserShare{
		FileID: t.FileID, FolderID: t.FolderID,
		UserID: toUser, CompanyID: toCompany, CanEdit: canEdit,
	}
	if err := s.repo.UpsertUserShare(ctx, share); err != nil {
		return nil, err
	}
	// Адресат должен увидеть появившееся у него сразу.
	if toUser != nil {
		s.bus.Publish(ctx, "drive:shared", rooms(*toUser), share)
	}
	return share, nil
}

func (s *Service) RevokeShare(ctx context.Context, userID, id int64) error {
	return s.repo.DeleteUserShare(ctx, id, userID)
}

// ByCode — файл или папка по публичной ссылке (без авторизации).
func (s *Service) ByCode(ctx context.Context, code string) (*domain.File, *domain.Folder, error) {
	share, err := s.repo.GetShareByCode(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	if share == nil {
		return nil, nil, domain.ErrNotFound
	}
	if share.FileID != nil {
		file, err := s.repo.GetFile(ctx, *share.FileID)
		if err != nil || file == nil || file.DeletedAt != nil {
			return nil, nil, domain.ErrNotFound
		}
		file.URL = fileURL(file.Key)
		return file, nil, nil
	}
	folder, err := s.repo.GetFolder(ctx, *share.FolderID)
	if err != nil || folder == nil || folder.DeletedAt != nil {
		return nil, nil, domain.ErrNotFound
	}
	return nil, folder, nil
}

// SharedListing — содержимое папки, открытой публичной ссылкой.
func (s *Service) SharedListing(ctx context.Context, code string) (*domain.Listing, error) {
	_, folder, err := s.ByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, domain.ErrNotFound
	}
	folders, err := s.repo.ListFolders(ctx, folder.OwnerID, &folder.ID, false)
	if err != nil {
		return nil, err
	}
	files, err := s.repo.ListFiles(ctx, domain.ListFilter{OwnerID: folder.OwnerID, FolderID: &folder.ID})
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		f.URL = fileURL(f.Key)
	}
	return &domain.Listing{Folder: folder, Folders: folders, Files: files, Path: []*domain.Folder{}}, nil
}

// SharedDownload — скачивание по публичной ссылке на файл.
func (s *Service) SharedDownload(ctx context.Context, code string) (*domain.File, []byte, error) {
	file, _, err := s.ByCode(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	if file == nil {
		return nil, nil, domain.ErrNotFound
	}
	data, err := s.files.Open(file.Key)
	if err != nil {
		return nil, nil, domain.ErrNotFound
	}
	return file, data, nil
}

func (s *Service) requireOwnTarget(ctx context.Context, userID int64, t Target) error {
	switch {
	case t.FileID != nil:
		_, err := s.requireOwnFile(ctx, userID, *t.FileID)
		return err
	case t.FolderID != nil:
		_, err := s.requireOwnFolder(ctx, userID, *t.FolderID)
		return err
	default:
		return domain.ErrValidation
	}
}

// SearchUsers — выбор адресата доступа.
func (s *Service) SearchUsers(ctx context.Context, query string, limit int) ([]*domain.User, error) {
	if query == "" {
		return []*domain.User{}, nil
	}
	return s.users.SearchUsers(ctx, query, limit)
}
