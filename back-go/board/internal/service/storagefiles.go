package service

import (
	"context"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/board/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
)

/* Раздел «Настройки → Хранилище»: биллинг спрашивает владельца файлов, что у
   него ещё живо, и просит удалить выбранное.

   У доски два вида файлов: картинки внутри сцены и миниатюра плитки. Удаление
   картинки вырезает объект из сцены — иначе на холсте осталась бы пустая
   рамка. Доски личные: companyIDs здесь не участвуют. */

func (s *Service) ListStorageFiles(ctx context.Context, userID int64, _ []int64) ([]storagefiles.File, error) {
	boards, err := s.repo.BoardScenesOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := []storagefiles.File{}
	for _, b := range boards {
		title := b.Title
		if title == "" {
			title = "Без названия"
		}
		file := storagefiles.File{
			Kind: "board", ID: strconv.FormatInt(b.ID, 10),
			Title: "Доска: " + title, CreatedAt: b.CreatedAt,
		}
		for _, key := range domain.SceneImageKeys(b.Scene) {
			img := file
			img.Key = key
			out = append(out, img)
		}
		if b.PreviewPath != "" {
			preview := file
			preview.Key, preview.Name = b.PreviewPath, "Миниатюра доски"
			out = append(out, preview)
		}
	}
	return out, nil
}

// DeleteStorageFiles — вырезать картинки из сцен, снять миниатюры и удалить
// сами файлы.
func (s *Service) DeleteStorageFiles(ctx context.Context, userID int64, _ []int64, keys []string) ([]string, error) {
	boards, err := s.repo.BoardScenesOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	wanted := make(map[string]bool, len(keys))
	for _, k := range keys {
		wanted[k] = true
	}

	deleted := []string{}
	for _, b := range boards {
		touched := false
		if scene, changed := domain.SceneWithoutImages(b.Scene, keys); changed {
			if err := s.repo.UpdateBoardScene(ctx, b.ID, scene, domain.SceneText(scene)); err != nil {
				return nil, err
			}
			left := map[string]bool{}
			for _, k := range domain.SceneImageKeys(scene) {
				left[k] = true
			}
			for _, k := range domain.SceneImageKeys(b.Scene) {
				if !left[k] {
					deleted = append(deleted, k)
				}
			}
			touched = true
		}
		if b.PreviewPath != "" && wanted[b.PreviewPath] {
			if err := s.repo.SetBoardPreview(ctx, b.ID, ""); err != nil {
				return nil, err
			}
			deleted = append(deleted, b.PreviewPath)
			touched = true
		}
		if touched {
			s.bus.Publish(ctx, "board:updated", s.boardRooms(ctx, b.ID, userID), map[string]any{"id": b.ID})
		}
	}
	if len(deleted) > 0 {
		s.files.Remove(deleted)
	}
	return deleted, nil
}

var _ storagefiles.Owner = (*Service)(nil)
