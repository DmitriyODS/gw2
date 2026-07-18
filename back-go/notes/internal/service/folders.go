package service

import (
	"context"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// FolderTree — папки раздела: свои (плоско, клиент строит дерево) и расшаренные
// мне «корни» (роль корней раздела «Поделились со мной»).
type FolderTree struct {
	Folders []*domain.Folder `json:"folders"`
	Shared  []*domain.Folder `json:"shared"`
}

func (s *Service) ListFolders(ctx context.Context, userID int64) (*FolderTree, error) {
	own, err := s.repo.ListFolders(ctx, userID)
	if err != nil {
		return nil, err
	}
	shared, err := s.repo.ListSharedRootFolders(ctx, userID, s.companyIDs(ctx, userID))
	if err != nil {
		return nil, err
	}
	return &FolderTree{Folders: own, Shared: shared}, nil
}

// FolderChildren — подпапки папки (для навигации по расшаренному поддереву) +
// my_access к самой папке. Доступ: своя или расшаренная мне/предку.
type FolderChildren struct {
	Folders  []*domain.Folder `json:"folders"`
	MyAccess string           `json:"my_access"`
}

func (s *Service) FolderChildren(ctx context.Context, userID, folderID int64) (*FolderChildren, error) {
	_, access, err := s.requireFolderReadable(ctx, userID, folderID)
	if err != nil {
		return nil, err
	}
	children, err := s.repo.ListChildFolders(ctx, folderID)
	if err != nil {
		return nil, err
	}
	if access != domain.AccessOwner {
		for _, c := range children {
			c.MyAccess = access
		}
	}
	return &FolderChildren{Folders: children, MyAccess: access}, nil
}

func (s *Service) CreateFolder(ctx context.Context, userID int64, name, color string, parentID *int64) (*domain.Folder, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrNameRequired
	}
	if color != "" && !domain.NoteColors[color] {
		return nil, domain.ErrBadColor
	}
	if parentID != nil {
		if _, err := s.requireFolderOwned(ctx, userID, *parentID); err != nil {
			return nil, err
		}
	}
	pos, err := s.repo.NextFolderPosition(ctx, userID, parentID)
	if err != nil {
		return nil, err
	}
	f := &domain.Folder{OwnerID: userID, ParentID: parentID, Name: name, Color: color, Position: pos}
	if err := s.repo.CreateFolder(ctx, f); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "note_folder:created", []string{userRoom(userID)}, folderPayload(f))
	return f, nil
}

func (s *Service) UpdateFolder(ctx context.Context, userID, id int64, name, color string) (*domain.Folder, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrNameRequired
	}
	if color != "" && !domain.NoteColors[color] {
		return nil, domain.ErrBadColor
	}
	f, err := s.requireFolderOwned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateFolder(ctx, id, name, color); err != nil {
		return nil, err
	}
	f.Name, f.Color = name, color
	s.publishFolder(ctx, "note_folder:updated", f)
	return f, nil
}

// MoveFolder — сменить родителя папки (parentID nil — в корень); только владелец.
// Защита от цикла: новый родитель не может быть самой папкой или её потомком.
func (s *Service) MoveFolder(ctx context.Context, userID, id int64, parentID *int64) (*domain.Folder, error) {
	f, err := s.requireFolderOwned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if parentID != nil {
		if *parentID == id {
			return nil, domain.ErrFolderCycle
		}
		if _, err := s.requireFolderOwned(ctx, userID, *parentID); err != nil {
			return nil, err
		}
		descendant, err := s.repo.IsDescendant(ctx, *parentID, id)
		if err != nil {
			return nil, err
		}
		if descendant {
			return nil, domain.ErrFolderCycle
		}
	}
	if err := s.repo.MoveFolder(ctx, id, parentID); err != nil {
		return nil, err
	}
	f.ParentID = parentID
	s.publishFolder(ctx, "note_folder:updated", f)
	return f, nil
}

// DeleteFolder — удалить папку; её прямые дети (подпапки и заметки) переезжают в
// родителя (Google-Drive-подобно: содержимое не пропадает). Только владелец.
func (s *Service) DeleteFolder(ctx context.Context, userID, id int64) error {
	f, err := s.requireFolderOwned(ctx, userID, id)
	if err != nil {
		return err
	}
	rooms := s.folderRooms(ctx, id, userID)
	if err := s.repo.ReparentChildren(ctx, id, f.ParentID); err != nil {
		return err
	}
	if err := s.repo.DeleteFolder(ctx, id); err != nil {
		return err
	}
	s.bus.Publish(ctx, "note_folder:deleted", rooms,
		map[string]any{"id": id, "owner_id": userID, "parent_id": f.ParentID})
	return nil
}

// CopyFolder — глубокая копия папки со всем поддеревом и заметками (в тот же
// родитель); только владелец.
func (s *Service) CopyFolder(ctx context.Context, userID, id int64) (*domain.Folder, error) {
	f, err := s.requireFolderOwned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	newID, err := s.repo.CopyFolderTree(ctx, userID, id, f.ParentID)
	if err != nil {
		return nil, err
	}
	cp, err := s.repo.GetFolder(ctx, newID)
	if err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "note_folder:created", []string{userRoom(userID)}, folderPayload(cp))
	return cp, nil
}

func folderPayload(f *domain.Folder) map[string]any {
	p := map[string]any{
		"id": f.ID, "owner_id": f.OwnerID, "parent_id": f.ParentID, "name": f.Name,
		"color": f.Color, "position": f.Position, "notes_count": f.NotesCount,
		"shared_by_me": f.SharedByMe, "created_at": f.CreatedAt, "updated_at": f.UpdatedAt,
	}
	if f.OwnerName != "" {
		p["owner_name"] = f.OwnerName
		p["owner_avatar"] = f.OwnerAvatar
	}
	if f.MyAccess != "" {
		p["my_access"] = f.MyAccess
	}
	return p
}

func (s *Service) publishFolder(ctx context.Context, event string, f *domain.Folder) {
	s.bus.Publish(ctx, event, s.folderRooms(ctx, f.ID, f.OwnerID), folderPayload(f))
}
