package service

import (
	"context"
	"encoding/json"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// collabKinds — допустимые типы collab-событий совместного редактирования.
var collabKinds = map[string]bool{"join": true, "leave": true, "cursor": true, "doc": true}

// Collab — лёгкий броадкаст совместного редактирования: НИЧЕГО не сохраняет в
// БД, только публикует note_collab:<kind> в комнаты владельца и всех адресатов
// (адресаты — открывшие документ, см. collabRooms). Доступ — владелец или
// адресат; kind=doc требует права правки (владелец/can_edit).
//
// Горячий путь (cursor/doc идут на каждое действие): ФИО отправителя кладётся
// в payload ТОЛЬКО на join — клиент кэширует его по user_id; cursor/leave/doc
// поле fio не несут, лишний запрос в users на каждое событие не делается.
func (s *Service) Collab(ctx context.Context, userID, noteID int64, kind string, cursor *domain.CollabCursor, doc json.RawMessage, title *string) error {
	if !collabKinds[kind] {
		return domain.ErrBadCollabKind
	}
	ownerID, access, err := s.collabAccess(ctx, userID, noteID, kind)
	if err != nil {
		return err
	}
	if kind == "doc" && access == domain.AccessView {
		return domain.ErrMemberReadOnly
	}
	payload := map[string]any{"note_id": noteID, "user_id": userID}
	if kind == "join" {
		if u, err := s.users.GetUser(ctx, userID); err == nil && u != nil {
			payload["fio"] = u.FIO
		}
	}
	if cursor != nil {
		payload["cursor"] = cursor
	}
	if doc != nil {
		clean, err := domain.SanitizeDoc(doc)
		if err != nil {
			return domain.ErrBadDoc
		}
		payload["doc"] = clean
	}
	// Название — часть live-правки (kind=doc, то же право): редактор шлёт его
	// вместе с документом, чтобы у соавторов заголовок менялся в реальном
	// времени, а не только после PATCH-сохранения.
	if title != nil && kind == "doc" {
		payload["title"] = *title
	}
	s.bus.Publish(ctx, "note_collab:"+kind, s.collabRooms(ctx, userID, noteID, ownerID, kind), payload)
	return nil
}

// collabAccess — доступ отправителя кадра. join проверяет честно и запоминает
// результат на окно зрителя, частые кадры (курсор, живые правки) берут
// запомненный: после отзыва шары рассылка прекращается не позже collab.TTL.
func (s *Service) collabAccess(ctx context.Context, userID, docID int64, kind string) (int64, string, error) {
	if s.viewers != nil && kind != "join" {
		if access, ownerID, ok := s.viewers.CachedAccess(ctx, docID, userID); ok {
			return ownerID, access, nil
		}
	}
	doc, access, err := s.requireReadable(ctx, userID, docID)
	if err != nil {
		return 0, "", err
	}
	if s.viewers != nil {
		s.viewers.RememberAccess(ctx, docID, userID, doc.OwnerID, access)
	}
	return doc.OwnerID, access, nil
}

// collabRooms — адресаты события совместной работы: открывшие документ, кроме
// самого отправителя. Реестр зрителей недоступен — вся аудитория, как раньше:
// лишние кадры лучше потерянной правки.
func (s *Service) collabRooms(ctx context.Context, userID, docID, ownerID int64, kind string) []string {
	if s.viewers == nil {
		return s.noteRooms(ctx, docID, ownerID)
	}
	var err error
	if kind == "leave" {
		err = s.viewers.Leave(ctx, docID, userID)
	} else {
		err = s.viewers.Touch(ctx, docID, userID)
	}
	var ids []int64
	if err == nil {
		ids, err = s.viewers.List(ctx, docID)
	}
	if err != nil {
		s.log.Warn("collab.viewers_failed", "doc_id", docID, "error", err)
		return s.noteRooms(ctx, docID, ownerID)
	}
	rooms := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != userID {
			rooms = append(rooms, userRoom(id))
		}
	}
	return rooms
}
