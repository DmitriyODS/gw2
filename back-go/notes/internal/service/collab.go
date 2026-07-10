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
// (включая отправителя — клиент отфильтрует по user_id). Доступ — владелец или
// адресат; kind=doc требует права правки (владелец/can_edit).
//
// Горячий путь (cursor/doc идут на каждое действие): ФИО отправителя кладётся
// в payload ТОЛЬКО на join — клиент кэширует его по user_id; cursor/leave/doc
// поле fio не несут, лишний запрос в users на каждое событие не делается.
func (s *Service) Collab(ctx context.Context, userID, noteID int64, kind string, cursor *domain.CollabCursor, doc json.RawMessage, title *string) error {
	if !collabKinds[kind] {
		return domain.ErrBadCollabKind
	}
	n, access, err := s.requireReadable(ctx, userID, noteID)
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
		payload["doc"] = doc
	}
	// Название — часть live-правки (kind=doc, то же право): редактор шлёт его
	// вместе с документом, чтобы у соавторов заголовок менялся в реальном
	// времени, а не только после PATCH-сохранения.
	if title != nil && kind == "doc" {
		payload["title"] = *title
	}
	s.bus.Publish(ctx, "note_collab:"+kind, s.noteRooms(ctx, noteID, n.OwnerID), payload)
	return nil
}
