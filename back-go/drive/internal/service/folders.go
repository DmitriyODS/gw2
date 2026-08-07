package service

import (
	"context"
	"slices"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/domain"
)

/* Папки диска. Дерево строит parent_id, а доступ к чужой папке считает
   репозиторий подъёмом по нему же — здесь только правила. */

// Browse — содержимое папки: подпапки, файлы и путь до корня.
func (s *Service) Browse(ctx context.Context, userID int64, f domain.ListFilter) (*domain.Listing, error) {
	out := &domain.Listing{Path: []*domain.Folder{}}

	// Чужая папка открывается, только если её открыли мне (каскад по дереву).
	ownerID := userID
	if f.FolderID != nil {
		folder, err := s.repo.GetFolder(ctx, *f.FolderID)
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
		if access == "" {
			return nil, domain.ErrForbidden
		}
		folder.MyAccess = access
		out.Folder = folder
		ownerID = folder.OwnerID
		if out.Path, err = s.repo.FolderPath(ctx, folder.ID); err != nil {
			return nil, err
		}
	}

	f.OwnerID = ownerID
	folders, err := s.repo.ListFolders(ctx, ownerID, f.FolderID, f.Trash)
	if err != nil {
		return nil, err
	}
	// Поиск ищет и папки — по всему диску, а не в текущей. «Избранное» и
	// «недавние» — про файлы, папки там не показываются.
	switch {
	case f.Search != "":
		if folders, err = s.repo.SearchFolders(ctx, ownerID, f.Search); err != nil {
			return nil, err
		}
	case f.Starred, f.Recent:
		folders = []*domain.Folder{}
	}
	files, err := s.repo.ListFiles(ctx, f)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		file.URL = fileURL(file.Key)
	}
	out.Folders, out.Files = folders, files
	s.markShared(ctx, out.Folders, out.Files)
	return out, nil
}

// markShared — значок «доступ открыт» на плитках владельца (одним запросом).
func (s *Service) markShared(ctx context.Context, folders []*domain.Folder, files []*domain.File) {
	fileIDs := make([]int64, 0, len(files))
	folderIDs := make([]int64, 0, len(folders))
	for _, f := range files {
		fileIDs = append(fileIDs, f.ID)
	}
	for _, f := range folders {
		folderIDs = append(folderIDs, f.ID)
	}
	sharedFiles, sharedFolders, err := s.repo.SharedByMe(ctx, fileIDs, folderIDs)
	if err != nil {
		s.log.Warn("drive.shared_flags_failed", "error", err)
		return
	}
	for _, f := range files {
		f.Shared = sharedFiles[f.ID]
	}
	for _, f := range folders {
		f.Shared = sharedFolders[f.ID]
	}
}

func (s *Service) CreateFolder(ctx context.Context, userID int64, name string, parentID *int64) (*domain.Folder, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrValidation
	}
	// Вкладывать можно только в свою папку (или в свой корень).
	if parentID != nil {
		if _, err := s.requireOwnFolder(ctx, userID, *parentID); err != nil {
			return nil, err
		}
	}
	folder := &domain.Folder{OwnerID: userID, ParentID: parentID, Name: name}
	if err := s.repo.CreateFolder(ctx, folder); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "drive_folder:created", rooms(userID), folder)
	return folder, nil
}

func (s *Service) RenameFolder(ctx context.Context, userID, id int64, name, color string) (*domain.Folder, error) {
	folder, err := s.requireOwnFolder(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrValidation
	}
	if err := s.repo.RenameFolder(ctx, id, name, color); err != nil {
		return nil, err
	}
	folder.Name, folder.Color = name, color
	s.bus.Publish(ctx, "drive_folder:updated", rooms(userID), folder)
	return folder, nil
}

// MoveFolder — перенос в другую свою папку. Папку нельзя положить в саму себя
// или в собственного потомка: поддерево оторвалось бы от дерева и стало
// недостижимым (в БД такая петля выглядит как отдельный «остров»).
func (s *Service) MoveFolder(ctx context.Context, userID, id int64, parentID *int64) (*domain.Folder, error) {
	folder, err := s.requireOwnFolder(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if parentID != nil {
		if _, err := s.requireOwnFolder(ctx, userID, *parentID); err != nil {
			return nil, err
		}
		subtree, err := s.repo.FolderSubtree(ctx, id)
		if err != nil {
			return nil, err
		}
		if slices.Contains(subtree, *parentID) {
			return nil, domain.ErrFolderCycle
		}
	}
	if err := s.repo.MoveFolder(ctx, id, parentID); err != nil {
		return nil, err
	}
	folder.ParentID = parentID
	s.bus.Publish(ctx, "drive_folder:updated", rooms(userID), folder)
	return folder, nil
}

// TrashFolder — папка уезжает в корзину вместе со всем поддеревом и файлами:
// человек удаляет папку целиком, а не её верхний уровень.
func (s *Service) TrashFolder(ctx context.Context, userID, id int64, deleted bool) error {
	if _, err := s.requireOwnFolder(ctx, userID, id); err != nil {
		return err
	}
	subtree, err := s.repo.FolderSubtree(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.SetFilesDeletedByFolders(ctx, subtree, deleted); err != nil {
		return err
	}
	if err := s.repo.SetFoldersDeleted(ctx, subtree, deleted); err != nil {
		return err
	}
	event := "drive_folder:trashed"
	if !deleted {
		event = "drive_folder:restored"
	}
	s.bus.Publish(ctx, event, rooms(userID), map[string]any{"id": id})
	return nil
}

// PurgeFolder — окончательное удаление из корзины вместе с файлами и их
// объектами в хранилище (место возвращается в квоту).
func (s *Service) PurgeFolder(ctx context.Context, userID, id int64) error {
	if _, err := s.requireOwnFolder(ctx, userID, id); err != nil {
		return err
	}
	subtree, err := s.repo.FolderSubtree(ctx, id)
	if err != nil {
		return err
	}
	files, err := s.repo.FilesOfFolders(ctx, subtree)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(files))
	ids := make([]int64, 0, len(files))
	for _, f := range files {
		keys = append(keys, f.Key)
		ids = append(ids, f.ID)
	}
	if err := s.repo.DeleteFiles(ctx, ids); err != nil {
		return err
	}
	if err := s.repo.DeleteFolders(ctx, subtree); err != nil {
		return err
	}
	if len(keys) > 0 {
		s.files.RemoveFor(ctx, userID, 0, keys)
	}
	s.bus.Publish(ctx, "drive_folder:deleted", rooms(userID), map[string]any{"id": id})
	return nil
}

// requireOwnFolder — папка существует и принадлежит мне. Правка структуры —
// дело владельца: адресату шаринга открыто содержимое, а не дерево.
func (s *Service) requireOwnFolder(ctx context.Context, userID, id int64) (*domain.Folder, error) {
	folder, err := s.repo.GetFolder(ctx, id)
	if err != nil {
		return nil, err
	}
	if folder == nil {
		return nil, domain.ErrNotFound
	}
	if folder.OwnerID != userID {
		return nil, domain.ErrForbidden
	}
	return folder, nil
}

func (s *Service) folderAccess(ctx context.Context, folder *domain.Folder, userID int64) (string, error) {
	if folder.OwnerID == userID {
		return domain.AccessOwner, nil
	}
	return s.repo.FolderAccess(ctx, folder.ID, userID, s.companiesOf(ctx, userID))
}

// fileURL — адрес объекта для клиента. Наружу отдаём именно его: клиенты
// работают с /uploads/<key>, а сам ключ — деталь хранилища.
func fileURL(key string) string {
	if key == "" {
		return ""
	}
	return "/uploads/" + key
}
