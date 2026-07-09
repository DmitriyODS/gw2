package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// emptyDoc — документ новой заметки: один пустой параграф (валидный корень
// TipTap, редактор открывается сразу с курсором).
var emptyDoc = json.RawMessage(`{"type":"doc","content":[{"type":"paragraph"}]}`)

// ListNotes — плитки владельца: по группе (0 — все) и сквозному поиску.
func (s *Service) ListNotes(ctx context.Context, userID, groupID int64, search string) ([]*domain.Note, error) {
	return s.repo.ListNotes(ctx, domain.NoteListFilter{
		OwnerID: userID, GroupID: groupID, Search: strings.TrimSpace(search),
	})
}

// GetNote — полная заметка владельца (с doc).
func (s *Service) GetNote(ctx context.Context, userID, id int64) (*domain.Note, error) {
	return s.requireOwned(ctx, userID, id)
}

func (s *Service) CreateNote(ctx context.Context, userID int64, title string) (*domain.Note, error) {
	n := &domain.Note{OwnerID: userID, Title: title, Doc: emptyDoc, GroupIDs: []int64{}}
	if err := s.repo.CreateNote(ctx, n); err != nil {
		return nil, err
	}
	s.publishNote(ctx, "note:created", n)
	return n, nil
}

// UpdateNote — частичная правка: nil-поля не меняются. При правке doc сервер
// пересчитывает text_content (поиск и txt-экспорт всегда согласованы с doc).
// Color — цвет плитки ('' — сбросить); по edit-ссылке не правится (личный стиль).
func (s *Service) UpdateNote(ctx context.Context, userID, id int64, title, color *string, doc json.RawMessage) (*domain.Note, error) {
	if color != nil && *color != "" && !domain.NoteColors[*color] {
		return nil, domain.ErrBadColor
	}
	n, err := s.requireOwned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if color != nil {
		n.Color = *color
	}
	return s.applyUpdate(ctx, n, title, doc)
}

// applyUpdate — общая запись правки (владелец и edit-ссылка).
func (s *Service) applyUpdate(ctx context.Context, n *domain.Note, title *string, doc json.RawMessage) (*domain.Note, error) {
	if title != nil {
		n.Title = *title
	}
	if doc != nil {
		n.Doc = doc
		n.TextContent = domain.DocText(doc)
	}
	if err := s.repo.UpdateNote(ctx, n); err != nil {
		return nil, err
	}
	s.publishNote(ctx, "note:updated", n)
	return n, nil
}

// DeleteNote — удаление заметки вместе с её картинками в хранилище.
func (s *Service) DeleteNote(ctx context.Context, userID, id int64) error {
	n, err := s.requireOwned(ctx, userID, id)
	if err != nil {
		return err
	}
	keys := domain.DocFileKeys(n.Doc)
	if err := s.repo.DeleteNote(ctx, id); err != nil {
		return err
	}
	if len(keys) > 0 {
		s.files.Remove(keys)
	}
	s.bus.Publish(ctx, "note:deleted", []string{userRoom(userID)}, map[string]any{"id": id, "owner_id": userID})
	return nil
}

// SetGroups — полная замена групп заметки; чужие/несуществующие группы молча
// отбрасываются.
func (s *Service) SetGroups(ctx context.Context, userID, noteID int64, groupIDs []int64) (*domain.Note, error) {
	n, err := s.requireOwned(ctx, userID, noteID)
	if err != nil {
		return nil, err
	}
	owned, err := s.repo.OwnedGroupIDs(ctx, userID, groupIDs)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetNoteGroups(ctx, noteID, owned); err != nil {
		return nil, err
	}
	n.GroupIDs = owned
	s.publishNote(ctx, "note:updated", n)
	return n, nil
}

// Upload — картинка редактора: только владелец заметки; клиенту возвращается
// готовый путь /uploads/<key> для вставки в документ.
func (s *Service) Upload(ctx context.Context, userID, noteID int64, fileName string, data []byte) (string, error) {
	if _, err := s.requireOwned(ctx, userID, noteID); err != nil {
		return "", err
	}
	key, err := s.files.Save(fileName, data)
	if err != nil {
		return "", err
	}
	return "/uploads/" + key, nil
}
