package service

import (
	"context"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
)

const (
	maxFolders          = 20
	maxFolderTitleRunes = 64
)

func errFolderNotFound() *domain.Error {
	return domain.NewError("FOLDER_NOT_FOUND", "Папка не найдена", 404)
}

// normalizeFolderInput валидирует и чистит поля папки.
func normalizeFolderInput(in dto.FolderInput) (dto.FolderInput, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return in, domain.NewError("FOLDER_TITLE_REQUIRED", "Введите название папки", 400)
	}
	if len([]rune(in.Title)) > maxFolderTitleRunes {
		return in, domain.NewError("FOLDER_TITLE_TOO_LONG", "Слишком длинное название папки", 400)
	}
	if in.Emoji != nil {
		e := strings.TrimSpace(*in.Emoji)
		if e == "" {
			in.Emoji = nil
		} else {
			in.Emoji = &e
		}
	}
	return in, nil
}

// ListFolders — папки пользователя (по порядку).
func (s *Service) ListFolders(ctx context.Context, userID int64) ([]*dto.Folder, error) {
	list, err := s.repo.ListFolders(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dto.NewFolders(list), nil
}

// CreateFolder — новая папка (в конец списка). Состав задаётся сразу, если
// передан ConversationIDs.
func (s *Service) CreateFolder(ctx context.Context, userID int64, in dto.FolderInput) (*dto.Folder, error) {
	in, err := normalizeFolderInput(in)
	if err != nil {
		return nil, err
	}
	n, err := s.repo.CountFolders(ctx, userID)
	if err != nil {
		return nil, err
	}
	if n >= maxFolders {
		return nil, domain.NewError("FOLDER_LIMIT", "Достигнут предел числа папок", 409)
	}

	f := &domain.Folder{
		OwnerID:         userID,
		Title:           in.Title,
		Emoji:           in.Emoji,
		IncludePersonal: in.IncludePersonal,
		IncludeGroups:   in.IncludeGroups,
		IncludeUnread:   in.IncludeUnread,
	}
	var id int64
	err = s.repo.RunInTx(ctx, func(ctx context.Context) error {
		var e error
		if id, e = s.repo.CreateFolder(ctx, f); e != nil {
			return e
		}
		if in.ConversationIDs != nil {
			return s.repo.SetFolderItems(ctx, userID, id, *in.ConversationIDs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.folderByID(ctx, userID, id)
}

// UpdateFolder — правка полей и (если передан) состава папки.
func (s *Service) UpdateFolder(ctx context.Context, userID, folderID int64, in dto.FolderInput) (*dto.Folder, error) {
	in, err := normalizeFolderInput(in)
	if err != nil {
		return nil, err
	}
	f := &domain.Folder{
		ID:              folderID,
		OwnerID:         userID,
		Title:           in.Title,
		Emoji:           in.Emoji,
		IncludePersonal: in.IncludePersonal,
		IncludeGroups:   in.IncludeGroups,
		IncludeUnread:   in.IncludeUnread,
	}
	err = s.repo.RunInTx(ctx, func(ctx context.Context) error {
		if e := s.repo.UpdateFolder(ctx, userID, f); e != nil {
			return e
		}
		if in.ConversationIDs != nil {
			return s.repo.SetFolderItems(ctx, userID, folderID, *in.ConversationIDs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.folderByID(ctx, userID, folderID)
}

func (s *Service) DeleteFolder(ctx context.Context, userID, folderID int64) error {
	if err := s.repo.DeleteFolder(ctx, userID, folderID); err != nil {
		return err
	}
	s.publishFoldersChanged(ctx, userID)
	return nil
}

func (s *Service) ReorderFolders(ctx context.Context, userID int64, orderedIDs []int64) error {
	if err := s.repo.ReorderFolders(ctx, userID, orderedIDs); err != nil {
		return err
	}
	s.publishFoldersChanged(ctx, userID)
	return nil
}

func (s *Service) AddFolderItem(ctx context.Context, userID, folderID, convID int64) error {
	if _, err := s.conversationForUser(ctx, convID, userID); err != nil {
		return err
	}
	if err := s.repo.AddFolderItem(ctx, userID, folderID, convID); err != nil {
		return err
	}
	s.publishFoldersChanged(ctx, userID)
	return nil
}

func (s *Service) RemoveFolderItem(ctx context.Context, userID, folderID, convID int64) error {
	if err := s.repo.RemoveFolderItem(ctx, userID, folderID, convID); err != nil {
		return err
	}
	s.publishFoldersChanged(ctx, userID)
	return nil
}

// folderByID — свежая папка после мутации; попутно рассылает эхо на другие
// устройства владельца.
func (s *Service) folderByID(ctx context.Context, userID, folderID int64) (*dto.Folder, error) {
	list, err := s.repo.ListFolders(ctx, userID)
	if err != nil {
		return nil, err
	}
	s.pub.Publish(ctx, "folders:changed", rooms(userID), map[string]any{})
	for _, f := range list {
		if f.ID == folderID {
			return dto.NewFolder(f), nil
		}
	}
	return nil, errFolderNotFound()
}

func (s *Service) publishFoldersChanged(ctx context.Context, userID int64) {
	s.pub.Publish(ctx, "folders:changed", rooms(userID), map[string]any{})
}
