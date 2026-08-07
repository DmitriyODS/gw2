package service

import (
	"context"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/storagefiles"
)

/* Раздел «Настройки → Хранилище»: биллинг спрашивает владельца файлов, что у
   него ещё живо, и просит удалить выбранное.

   Картинки заметок лежат ссылками внутри документа TipTap, поэтому список
   собирается обходом документов владельца, а удаление вырезает узел из
   документа — иначе на его месте осталась бы битая картинка. Заметки личные:
   companyIDs здесь не участвуют. */

func (s *Service) ListStorageFiles(ctx context.Context, userID int64, _ []int64) ([]storagefiles.File, error) {
	notes, err := s.repo.NoteDocsOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := []storagefiles.File{}
	for _, n := range notes {
		title := n.Title
		if title == "" {
			title = "Без названия"
		}
		for _, key := range domain.DocFileKeys(n.Doc) {
			out = append(out, storagefiles.File{
				Key: key, Kind: "note", ID: strconv.FormatInt(n.ID, 10),
				Title: "Заметка: " + title, CreatedAt: n.CreatedAt,
			})
		}
	}
	return out, nil
}

// DeleteStorageFiles — вырезать картинки из документов и удалить сами файлы.
func (s *Service) DeleteStorageFiles(ctx context.Context, userID int64, _ []int64, keys []string) ([]string, error) {
	notes, err := s.repo.NoteDocsOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	deleted := []string{}
	for _, n := range notes {
		doc, changed := domain.DocWithoutFiles(n.Doc, keys)
		if !changed {
			continue
		}
		if err := s.repo.UpdateNoteDoc(ctx, n.ID, doc, domain.DocText(doc)); err != nil {
			return nil, err
		}
		// Что реально ушло из этой заметки — разница ключей до и после.
		left := map[string]bool{}
		for _, k := range domain.DocFileKeys(doc) {
			left[k] = true
		}
		for _, k := range domain.DocFileKeys(n.Doc) {
			if !left[k] {
				deleted = append(deleted, k)
			}
		}
		s.bus.Publish(ctx, "note:updated", s.noteRooms(ctx, n.ID, userID), map[string]any{"id": n.ID})
	}
	if len(deleted) > 0 {
		s.files.Remove(deleted)
	}
	return deleted, nil
}

var _ storagefiles.Owner = (*Service)(nil)
