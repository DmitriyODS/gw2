package service

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/domain"
)

/* Файлы диска. Объект кладётся в общее хранилище (pkg/storage) и сразу
   считается в квоте владельца: SaveFor не запишет файл сверх лимита. */

// Upload — загрузка файла в папку. Папка чужая — нужен доступ на правку.
func (s *Service) Upload(ctx context.Context, userID int64, name string, data []byte,
	mime string, folderID *int64) (*domain.File, error) {

	if len(data) == 0 {
		return nil, domain.ErrEmptyFile
	}
	if len(data) > domain.MaxFileSize {
		return nil, domain.ErrFileTooBig
	}
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" {
		name = "файл"
	}

	// Владелец файла — владелец папки: место тратит тот, у кого он лежит.
	ownerID := userID
	if folderID != nil {
		folder, err := s.repo.GetFolder(ctx, *folderID)
		if err != nil {
			return nil, err
		}
		if folder == nil || folder.DeletedAt != nil {
			return nil, domain.ErrNotFound
		}
		access, err := s.folderAccess(ctx, folder, userID)
		if err != nil {
			return nil, err
		}
		if !domain.AccessAtLeast(access, domain.AccessEdit) {
			return nil, domain.ErrForbidden
		}
		ownerID = folder.OwnerID
	}

	key, err := s.files.SaveFor(ctx, ownerID, 0, name, data)
	if err != nil {
		return nil, err // сверх квоты биллинг вернёт понятную ошибку
	}
	file := &domain.File{
		OwnerID: ownerID, FolderID: folderID, Name: name,
		Key: key, Mime: mime, Size: int64(len(data)),
	}
	if err := s.repo.CreateFile(ctx, file); err != nil {
		s.files.RemoveFor(ctx, ownerID, 0, []string{key}) // объект без записи — мусор
		return nil, err
	}
	file.URL = fileURL(key)
	s.bus.Publish(ctx, "drive_file:created", rooms(ownerID, userID), file)
	return file, nil
}

// Get — файл с проверкой доступа (просмотр и скачивание).
func (s *Service) Get(ctx context.Context, userID, id int64) (*domain.File, error) {
	file, access, err := s.fileWithAccess(ctx, userID, id, domain.AccessView)
	if err != nil {
		return nil, err
	}
	file.MyAccess = access
	file.URL = fileURL(file.Key)
	return file, nil
}

// Download — содержимое файла (скачивание и просмотр в разделе).
func (s *Service) Download(ctx context.Context, userID, id int64) (*domain.File, []byte, error) {
	file, _, err := s.fileWithAccess(ctx, userID, id, domain.AccessView)
	if err != nil {
		return nil, nil, err
	}
	data, err := s.files.Open(file.Key)
	if err != nil {
		return nil, nil, domain.ErrNotFound
	}
	return file, data, nil
}

func (s *Service) Rename(ctx context.Context, userID, id int64, name string) (*domain.File, error) {
	file, _, err := s.fileWithAccess(ctx, userID, id, domain.AccessEdit)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrValidation
	}
	if err := s.repo.RenameFile(ctx, id, name); err != nil {
		return nil, err
	}
	file.Name, file.URL = name, fileURL(file.Key)
	s.bus.Publish(ctx, "drive_file:updated", rooms(file.OwnerID, userID), file)
	return file, nil
}

// Move — перенос в другую СВОЮ папку (или в корень).
func (s *Service) Move(ctx context.Context, userID, id int64, folderID *int64) (*domain.File, error) {
	file, err := s.requireOwnFile(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if folderID != nil {
		if _, err := s.requireOwnFolder(ctx, userID, *folderID); err != nil {
			return nil, err
		}
	}
	if err := s.repo.MoveFile(ctx, id, folderID); err != nil {
		return nil, err
	}
	file.FolderID, file.URL = folderID, fileURL(file.Key)
	s.bus.Publish(ctx, "drive_file:updated", rooms(userID), file)
	return file, nil
}

func (s *Service) Star(ctx context.Context, userID, id int64, starred bool) (*domain.File, error) {
	file, err := s.requireOwnFile(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetFileStarred(ctx, id, starred); err != nil {
		return nil, err
	}
	file.Starred, file.URL = starred, fileURL(file.Key)
	s.bus.Publish(ctx, "drive_file:updated", rooms(userID), file)
	return file, nil
}

// Trash — в корзину и обратно. Место остаётся занятым: пока файл можно
// вернуть, он никуда не делся.
func (s *Service) Trash(ctx context.Context, userID, id int64, deleted bool) error {
	if _, err := s.requireOwnFile(ctx, userID, id); err != nil {
		return err
	}
	if err := s.repo.SetFilesDeleted(ctx, []int64{id}, deleted); err != nil {
		return err
	}
	event := "drive_file:trashed"
	if !deleted {
		event = "drive_file:restored"
	}
	s.bus.Publish(ctx, event, rooms(userID), map[string]any{"id": id})
	return nil
}

// Purge — окончательное удаление: запись, объект в хранилище и место в квоте.
func (s *Service) Purge(ctx context.Context, userID, id int64) error {
	file, err := s.requireOwnFile(ctx, userID, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteFiles(ctx, []int64{id}); err != nil {
		return err
	}
	s.files.RemoveFor(ctx, userID, 0, []string{file.Key})
	s.bus.Publish(ctx, "drive_file:deleted", rooms(userID), map[string]any{"id": id})
	return nil
}

// EmptyTrash — очистить корзину целиком.
func (s *Service) EmptyTrash(ctx context.Context, userID int64) (int, error) {
	files, err := s.repo.ListFiles(ctx, domain.ListFilter{OwnerID: userID, Trash: true})
	if err != nil {
		return 0, err
	}
	folders, err := s.repo.ListFolders(ctx, userID, nil, true)
	if err != nil {
		return 0, err
	}
	ids := make([]int64, 0, len(files))
	keys := make([]string, 0, len(files))
	for _, f := range files {
		ids = append(ids, f.ID)
		keys = append(keys, f.Key)
	}
	folderIDs := make([]int64, 0, len(folders))
	for _, f := range folders {
		folderIDs = append(folderIDs, f.ID)
	}
	if err := s.repo.DeleteFiles(ctx, ids); err != nil {
		return 0, err
	}
	if err := s.repo.DeleteFolders(ctx, folderIDs); err != nil {
		return 0, err
	}
	if len(keys) > 0 {
		s.files.RemoveFor(ctx, userID, 0, keys)
	}
	s.bus.Publish(ctx, "drive:trash-emptied", rooms(userID), map[string]any{"files": len(ids)})
	return len(ids), nil
}

// SharedWithMe — чужое, открытое мне лично или моей компании.
func (s *Service) SharedWithMe(ctx context.Context, userID int64) (*domain.Listing, error) {
	folders, files, err := s.repo.SharedWithMe(ctx, userID, s.companiesOf(ctx, userID))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		f.URL = fileURL(f.Key)
	}
	return &domain.Listing{Folders: folders, Files: files, Path: []*domain.Folder{}}, nil
}

func (s *Service) requireOwnFile(ctx context.Context, userID, id int64) (*domain.File, error) {
	file, err := s.repo.GetFile(ctx, id)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, domain.ErrNotFound
	}
	if file.OwnerID != userID {
		return nil, domain.ErrForbidden
	}
	return file, nil
}

// fileWithAccess — файл и мой уровень доступа к нему; меньше требуемого — отказ.
func (s *Service) fileWithAccess(ctx context.Context, userID, id int64, want string) (*domain.File, string, error) {
	file, err := s.repo.GetFile(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if file == nil {
		return nil, "", domain.ErrNotFound
	}
	access := domain.AccessOwner
	if file.OwnerID != userID {
		if access, err = s.repo.FileAccess(ctx, id, userID, s.companiesOf(ctx, userID)); err != nil {
			return nil, "", err
		}
	}
	if !domain.AccessAtLeast(access, want) {
		return nil, "", domain.ErrForbidden
	}
	return file, access, nil
}
