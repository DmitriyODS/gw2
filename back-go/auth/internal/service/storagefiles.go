package service

import (
	"context"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
)

/* Раздел «Настройки → Хранилище»: биллинг спрашивает владельца файлов, что у
   него ещё живо, и просит удалить выбранное. У authsvc это ровно один файл на
   человека — фото профиля; компании тут ни при чём. */

func (s *Service) ListStorageFiles(ctx context.Context, userID int64, _ []int64) ([]storagefiles.File, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil || user == nil || user.AvatarPath == nil || *user.AvatarPath == "" {
		return nil, err
	}
	return []storagefiles.File{{
		Key: *user.AvatarPath, Name: "Фото профиля", Kind: "avatar",
		ID: strconv.FormatInt(userID, 10), Title: "Фото профиля",
	}}, nil
}

// DeleteStorageFiles — снять фото профиля (дальше показывается identicon).
func (s *Service) DeleteStorageFiles(ctx context.Context, userID int64, _ []int64, keys []string) ([]string, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil || user == nil || user.AvatarPath == nil {
		return nil, err
	}
	current := *user.AvatarPath
	for _, key := range keys {
		if key != current {
			continue
		}
		s.avatars.Delete(current)
		if err := s.repo.UpdateFields(ctx, userID, map[string]any{"avatar_path": nil}); err != nil {
			return nil, err
		}
		// Журнал чистит сам биллинг по ответу, а вот его кэш лимитов —
		// наш: место освободилось прямо сейчас.
		s.billing.Invalidate(userID)
		return []string{current}, nil
	}
	return nil, nil
}

// Проверка контракта на этапе компиляции: сервер поднимает cmd.
var _ storagefiles.Owner = (*Service)(nil)
