package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// emptyDoc — документ новой заметки: один пустой параграф (валидный корень
// TipTap, редактор открывается сразу с курсором).
var emptyDoc = json.RawMessage(`{"type":"doc","content":[{"type":"paragraph"}]}`)

// ListNotes — плитки владельца: по группе (0 — все), сквозному поиску и
// архивности (архив — отдельный фильтр, в основной список не попадает).
func (s *Service) ListNotes(ctx context.Context, userID, groupID int64, search string, archived bool) ([]*domain.Note, error) {
	return s.repo.ListNotes(ctx, domain.NoteListFilter{
		OwnerID: userID, GroupID: groupID, Search: strings.TrimSpace(search), Archived: archived,
	})
}

// ListSharedNotes — чужие заметки, открытые пользователю адресно («поделились
// со мной»): плитки без doc, с владельцем и my_access (edit|view). Фильтры
// group_id/archived и закрепление не применяются — это личная организация
// владельца.
func (s *Service) ListSharedNotes(ctx context.Context, userID int64, search string) ([]*domain.Note, error) {
	return s.repo.ListSharedWithMe(ctx, userID, strings.TrimSpace(search))
}

// GetNote — полная заметка (с doc), доступная пользователю: своя или открытая
// адресно; my_access — owner | edit | view. Публичные ссылки (shares) в ответ
// не входят ни для кого — они отдаются отдельной владельческой ручкой
// GET /:id/shares, поэтому адресату не утекают.
func (s *Service) GetNote(ctx context.Context, userID, id int64) (*domain.Note, error) {
	n, access, err := s.requireReadable(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	n.MyAccess = access
	if access != domain.AccessOwner {
		// Группы — личная организация владельца, адресату не отдаются.
		n.GroupIDs = []int64{}
		if owner, err := s.users.GetUser(ctx, n.OwnerID); err == nil && owner != nil {
			n.OwnerName, n.OwnerAvatar = owner.FIO, owner.AvatarPath
		}
	}
	return n, nil
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
// Color (” — сбросить), Archived и Pinned (закрепление: true → pinned_at=now,
// false → сброс) правятся только владельцем — это личный стиль плитки; адресат
// с can_edit правит только title/doc (color/archived/pinned у него молча
// игнорируются, как и по edit-ссылке), адресат без can_edit — 403.
func (s *Service) UpdateNote(ctx context.Context, userID, id int64, u domain.NoteUpdate) (*domain.Note, error) {
	if u.Color != nil && *u.Color != "" && !domain.NoteColors[*u.Color] {
		return nil, domain.ErrBadColor
	}
	n, access, err := s.requireReadable(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	switch access {
	case domain.AccessOwner:
		if u.Color != nil {
			n.Color = *u.Color
		}
		if u.Archived != nil {
			n.Archived = *u.Archived
		}
		if u.Pinned != nil {
			if *u.Pinned {
				now := time.Now()
				n.PinnedAt = &now
			} else {
				n.PinnedAt = nil
			}
		}
	case domain.AccessEdit: // адресату — только title/doc
	default:
		return nil, domain.ErrMemberReadOnly
	}
	return s.applyUpdate(ctx, n, u.Title, u.Doc)
}

// applyUpdate — общая запись правки (владелец, адресат с can_edit,
// edit-ссылка). note:updated уходит владельцу и всем адресатам.
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
	s.bus.Publish(ctx, "note:updated", s.noteRooms(ctx, n.ID, n.OwnerID), notePayload(n))
	return n, nil
}

// DeleteNote — удаление заметки вместе с её картинками в хранилище.
// note:deleted уходит и адресатам — заметка пропадает у них вживую.
func (s *Service) DeleteNote(ctx context.Context, userID, id int64) error {
	n, err := s.requireOwned(ctx, userID, id)
	if err != nil {
		return err
	}
	rooms := s.noteRooms(ctx, id, userID) // до удаления — каскад чистит адресатов
	keys := domain.DocFileKeys(n.Doc)
	if err := s.repo.DeleteNote(ctx, id); err != nil {
		return err
	}
	if len(keys) > 0 {
		s.files.Remove(keys)
	}
	s.bus.Publish(ctx, "note:deleted", rooms, map[string]any{"id": id, "owner_id": userID})
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
