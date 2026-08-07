package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
)

/* Раздел «Настройки → Хранилище»: биллинг спрашивает владельца файлов, что у
   него ещё живо, и просит удалить выбранное. Вложения кросс-компанийны и
   принадлежат загрузившему, поэтому companyIDs здесь не участвуют. */

func (s *Service) ListStorageFiles(ctx context.Context, userID int64, _ []int64) ([]storagefiles.File, error) {
	return s.repo.ListStorageFiles(ctx, userID)
}

// DeleteStorageFiles — снять вложения с сообщений и удалить сами файлы.
// Сообщение остаётся: чистится место, а не переписка.
func (s *Service) DeleteStorageFiles(ctx context.Context, userID int64, _ []int64, keys []string) ([]string, error) {
	deleted, err := s.repo.DeleteStorageFiles(ctx, userID, keys)
	if err != nil {
		return nil, err
	}
	if len(deleted) > 0 {
		s.files.Remove(deleted)
	}
	return deleted, nil
}
