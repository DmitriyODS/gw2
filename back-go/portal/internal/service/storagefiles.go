package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
)

/* Раздел «Настройки → Хранилище»: биллинг спрашивает владельца файлов, что у
   него ещё живо, и просит удалить выбранное.

   Файлы портала принадлежат КОМПАНИИ, а платит за них её создатель, поэтому
   работаем по companyIDs — их присылает биллинг, он же знает создателей. */

func (s *Service) ListStorageFiles(ctx context.Context, _ int64, companyIDs []int64) ([]storagefiles.File, error) {
	return s.repo.ListStorageFiles(ctx, companyIDs)
}

// DeleteStorageFiles — снять вложения с публикаций и удалить сами файлы.
// Публикации остаются: освобождается место, а не стирается лента.
func (s *Service) DeleteStorageFiles(ctx context.Context, _ int64, companyIDs []int64, keys []string) ([]string, error) {
	deleted, err := s.repo.DeleteStorageFiles(ctx, companyIDs, keys)
	if err != nil {
		return nil, err
	}
	if len(deleted) > 0 {
		s.files.Remove(deleted)
	}
	return deleted, nil
}

var _ storagefiles.Owner = (*Service)(nil)
