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

// ListNotesParams — параметры выборки плиток раздела.
type ListNotesParams struct {
	FolderID  *int64
	FolderSet bool
	TagIDs    []int64
	Search    string
	Archived  bool
}

// ListNotes — плитки владельца по фильтру; при просмотре ЧУЖОЙ (расшаренной)
// папки — её заметки с owner/my_access. При глобальном поиске и включённом ИИ —
// семантический поиск по своим заметкам (fail-open на текстовый).
func (s *Service) ListNotes(ctx context.Context, userID, companyID int64, p ListNotesParams) ([]*domain.Note, error) {
	if p.Search != "" && !p.FolderSet && len(p.TagIDs) == 0 && s.aiEnabled() && companyID > 0 {
		if notes, ok := s.semanticNotes(ctx, userID, companyID, strings.TrimSpace(p.Search), p.Archived); ok {
			s.markSharedByMe(ctx, notes)
			return notes, nil
		}
	}
	f := domain.NoteListFilter{
		OwnerID: userID, FolderID: p.FolderID, FolderSet: p.FolderSet,
		TagIDs: p.TagIDs, Search: strings.TrimSpace(p.Search), Archived: p.Archived,
	}
	if p.FolderSet && p.FolderID != nil {
		fol, err := s.repo.GetFolder(ctx, *p.FolderID)
		if err != nil {
			return nil, err
		}
		if fol == nil {
			return nil, domain.ErrFolderNotFound
		}
		if fol.OwnerID != userID {
			_, access, err := s.requireFolderReadable(ctx, userID, *p.FolderID)
			if err != nil {
				return nil, err
			}
			f.OwnerID = 0 // заметки владельца папки
			f.Archived = false
			notes, err := s.repo.ListNotes(ctx, f)
			if err != nil {
				return nil, err
			}
			s.decorateShared(ctx, notes, fol.OwnerID, access)
			return notes, nil
		}
	}
	notes, err := s.repo.ListNotes(ctx, f)
	if err != nil {
		return nil, err
	}
	s.markSharedByMe(ctx, notes)
	if placed := s.recipientNotesFor(ctx, userID, p); placed != nil {
		notes = append(notes, placed...)
	}
	return notes, nil
}

// recipientNotesFor — чужие заметки, размещённые мной в текущем scope проводника
// (папка/корень) или в личном архиве. Только для «своих» видов без поиска/тегов
// (теги — личная организация владельца, поиск идёт своим путём).
func (s *Service) recipientNotesFor(ctx context.Context, userID int64, p ListNotesParams) []*domain.Note {
	if p.Search != "" || len(p.TagIDs) > 0 {
		return nil
	}
	var scope domain.RecipientScope
	var folderID *int64
	switch {
	case p.Archived && !p.FolderSet:
		scope = domain.RecipientArchive
	case p.FolderSet && !p.Archived && p.FolderID != nil:
		scope, folderID = domain.RecipientFolder, p.FolderID
	case p.FolderSet && !p.Archived && p.FolderID == nil:
		scope = domain.RecipientRoot
	default:
		return nil
	}
	placed, err := s.repo.ListRecipientNotes(ctx, userID, s.companyIDs(ctx, userID), scope, folderID)
	if err != nil {
		s.log.Warn("notes.recipient_notes_failed", "user", userID, "error", err)
		return nil
	}
	return placed
}

// ListSharedNotes — чужие заметки, доступные мне адресно или через расшаренную
// папку («поделились со мной»): плитки без doc, с владельцем и my_access.
func (s *Service) ListSharedNotes(ctx context.Context, userID int64, search string) ([]*domain.Note, error) {
	return s.repo.ListSharedWithMe(ctx, userID, s.companyIDs(ctx, userID), strings.TrimSpace(search))
}

// decorateShared — проставить owner и my_access плиткам чужой расшаренной папки.
func (s *Service) decorateShared(ctx context.Context, notes []*domain.Note, ownerID int64, access string) {
	if len(notes) == 0 {
		return
	}
	var name string
	var avatar *string
	if owner, err := s.users.GetUser(ctx, ownerID); err == nil && owner != nil {
		name, avatar = owner.FIO, owner.AvatarPath
	}
	for _, n := range notes {
		n.OwnerName, n.OwnerAvatar, n.MyAccess = name, avatar, access
		n.TagIDs = []int64{} // теги — личные метки владельца
	}
}

// markSharedByMe — проставить SharedByMe плиткам владельца (значок «расшарено»).
func (s *Service) markSharedByMe(ctx context.Context, notes []*domain.Note) {
	if len(notes) == 0 {
		return
	}
	ids := make([]int64, len(notes))
	for i, n := range notes {
		ids[i] = n.ID
	}
	shared, err := s.repo.SharedByMeNoteIDs(ctx, ids)
	if err != nil {
		return
	}
	for _, n := range notes {
		if shared[n.ID] {
			n.SharedByMe = true
		}
	}
}

// GetNote — полная заметка (с doc), доступная пользователю: своя или открытая
// шаром/папкой; my_access — owner | edit | view.
func (s *Service) GetNote(ctx context.Context, userID, id int64) (*domain.Note, error) {
	n, access, err := s.requireReadable(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	n.MyAccess = access
	if access != domain.AccessOwner {
		n.TagIDs = []int64{} // теги — личная организация владельца
		if owner, err := s.users.GetUser(ctx, n.OwnerID); err == nil && owner != nil {
			n.OwnerName, n.OwnerAvatar = owner.FIO, owner.AvatarPath
		}
	}
	return n, nil
}

// CreateNote — новая заметка (опционально в папке владельца).
func (s *Service) CreateNote(ctx context.Context, userID int64, title string, folderID *int64) (*domain.Note, error) {
	if err := s.checkOwnFolder(ctx, userID, folderID); err != nil {
		return nil, err
	}
	n := &domain.Note{OwnerID: userID, Title: title, Doc: emptyDoc, FolderID: folderID, TagIDs: []int64{}}
	if err := s.repo.CreateNote(ctx, n); err != nil {
		return nil, err
	}
	s.publishNote(ctx, "note:created", n)
	return n, nil
}

// UpdateNote — частичная правка: nil-поля не меняются. При правке doc сервер
// пересчитывает text_content. Color/Archived/Pinned — только владелец; Title/Doc
// — владелец, адресат с can_edit или edit-ссылка.
func (s *Service) UpdateNote(ctx context.Context, userID, id int64, u domain.NoteUpdate) (*domain.Note, error) {
	if u.Color != nil && *u.Color != "" && !domain.NoteColors[*u.Color] {
		return nil, domain.ErrBadColor
	}
	n, access, err := s.requireReadable(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	// Адресату — личный архив «только у меня» (даже при доступе только на чтение):
	// это моя организация, а не правка содержимого владельца.
	if access != domain.AccessOwner && u.Archived != nil {
		if err := s.repo.SetNoteRecipientArchived(ctx, userID, id, *u.Archived); err != nil {
			return nil, err
		}
		resp := *n // личный архив — не трогаем заметку владельца
		resp.Archived = *u.Archived
		return s.publishRecipientNote(ctx, userID, &resp, access), nil
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

// applyUpdate — общая запись правки (владелец, адресат с can_edit, edit-ссылка).
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
func (s *Service) DeleteNote(ctx context.Context, userID, id int64) error {
	n, err := s.requireOwned(ctx, userID, id)
	if err != nil {
		return err
	}
	rooms := s.noteRooms(ctx, id, userID) // до удаления — аудитория ещё цела
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

// MoveNote — разложить заметку по папкам (folderID nil — в корень). Владелец
// двигает саму заметку; адресат шаринга — раскладывает ЧУЖУЮ заметку по СВОИМ
// папкам через личный оверлей (у владельца ничего не меняется). Цель — всегда
// моя папка (checkOwnFolder).
func (s *Service) MoveNote(ctx context.Context, userID, id int64, folderID *int64) (*domain.Note, error) {
	n, access, err := s.requireReadable(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := s.checkOwnFolder(ctx, userID, folderID); err != nil {
		return nil, err
	}
	if access == domain.AccessOwner {
		if err := s.repo.MoveNote(ctx, id, folderID); err != nil {
			return nil, err
		}
		n.FolderID = folderID
		s.publishNote(ctx, "note:updated", n)
		return n, nil
	}
	if err := s.repo.SetNoteRecipientPlacement(ctx, userID, id, folderID); err != nil {
		return nil, err
	}
	resp := *n // не мутируем заметку владельца — оверлей чисто мой
	resp.FolderID = folderID
	resp.Archived = false
	return s.publishRecipientNote(ctx, userID, &resp, access), nil
}

// publishRecipientNote — оформить чужую заметку как размещённую мной (owner-поля,
// my_access, без личных тегов) и разослать событие только в мою комнату.
func (s *Service) publishRecipientNote(ctx context.Context, userID int64, n *domain.Note, access string) *domain.Note {
	s.decorateShared(ctx, []*domain.Note{n}, n.OwnerID, access)
	s.bus.Publish(ctx, "note:updated", []string{userRoom(userID)}, notePayload(n))
	return n
}

// CopyNote — дубликат заметки владельца (в той же папке, с тегами).
func (s *Service) CopyNote(ctx context.Context, userID, id int64) (*domain.Note, error) {
	src, err := s.requireOwned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	cp := &domain.Note{
		OwnerID: userID, FolderID: src.FolderID, Title: copyTitle(src.Title),
		Color: src.Color, Doc: src.Doc, TextContent: src.TextContent, TagIDs: []int64{},
	}
	if err := s.repo.CreateNote(ctx, cp); err != nil {
		return nil, err
	}
	if len(src.TagIDs) > 0 {
		if err := s.repo.SetNoteTags(ctx, cp.ID, src.TagIDs); err == nil {
			cp.TagIDs = src.TagIDs
		}
	}
	s.publishNote(ctx, "note:created", cp)
	return cp, nil
}

// SetTags — полная замена тегов заметки (только владелец); чужие/несуществующие
// теги молча отбрасываются.
func (s *Service) SetTags(ctx context.Context, userID, noteID int64, tagIDs []int64) (*domain.Note, error) {
	n, err := s.requireOwned(ctx, userID, noteID)
	if err != nil {
		return nil, err
	}
	owned, err := s.repo.OwnedTagIDs(ctx, userID, tagIDs)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetNoteTags(ctx, noteID, owned); err != nil {
		return nil, err
	}
	n.TagIDs = owned
	s.publishNote(ctx, "note:updated", n)
	return n, nil
}

// Upload — картинка редактора: владелец или адресат с правом правки.
func (s *Service) Upload(ctx context.Context, userID, noteID int64, fileName string, data []byte) (string, error) {
	_, access, err := s.requireReadable(ctx, userID, noteID)
	if err != nil {
		return "", err
	}
	if access != domain.AccessOwner && access != domain.AccessEdit {
		return "", domain.ErrMemberReadOnly
	}
	key, err := s.files.Save(fileName, data)
	if err != nil {
		return "", err
	}
	return "/uploads/" + key, nil
}

// checkOwnFolder — папка (если задана) принадлежит пользователю.
func (s *Service) checkOwnFolder(ctx context.Context, userID int64, folderID *int64) error {
	if folderID == nil {
		return nil
	}
	_, err := s.requireFolderOwned(ctx, userID, *folderID)
	return err
}

func copyTitle(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return "Копия"
	}
	return t + " (копия)"
}
