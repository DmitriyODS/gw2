package service

import (
	"context"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/drive/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
)

/* Раздел «Настройки → Хранилище»: биллинг спрашивает владельца файлов, что у
   него ещё живо, и просит удалить выбранное.

   Диск — самый прямолинейный владелец: файл и есть сущность. Корзина при этом
   считается живой — пока файл можно вернуть, место занято по-настоящему. */

func (s *Service) ListStorageFiles(ctx context.Context, userID int64, _ []int64) ([]storagefiles.File, error) {
	out := []storagefiles.File{}
	for _, trash := range []bool{false, true} {
		files, err := s.repo.ListFiles(ctx, domain.ListFilter{OwnerID: userID, Trash: trash})
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			title := "Диск"
			if trash {
				title = "Диск · корзина"
			}
			out = append(out, storagefiles.File{
				Key: f.Key, Name: f.Name, Kind: "drive",
				ID: strconv.FormatInt(f.ID, 10), Title: title, CreatedAt: f.CreatedAt,
			})
		}
	}
	return out, nil
}

// DeleteStorageFiles — удалить файлы диска насовсем (мимо корзины: человек уже
// выбрал их в разделе «Хранилище» именно чтобы освободить место).
func (s *Service) DeleteStorageFiles(ctx context.Context, userID int64, _ []int64, keys []string) ([]string, error) {
	wanted := make(map[string]bool, len(keys))
	for _, k := range keys {
		wanted[k] = true
	}
	ids := []int64{}
	deleted := []string{}
	for _, trash := range []bool{false, true} {
		files, err := s.repo.ListFiles(ctx, domain.ListFilter{OwnerID: userID, Trash: trash})
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if wanted[f.Key] {
				ids = append(ids, f.ID)
				deleted = append(deleted, f.Key)
			}
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	if err := s.repo.DeleteFiles(ctx, ids); err != nil {
		return nil, err
	}
	s.files.Remove(deleted) // место пересчитает сам биллинг
	s.bus.Publish(ctx, "drive:files-purged", rooms(userID), map[string]any{"ids": ids})
	return deleted, nil
}

var _ storagefiles.Owner = (*Service)(nil)
