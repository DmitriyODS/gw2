package service

import (
	"context"
	"encoding/json"

	"github.com/DmitriyODS/gw2/back-go/board/internal/domain"
)

// collabKinds — допустимые типы collab-событий совместного редактирования.
var collabKinds = map[string]bool{
	"join": true, "leave": true, "cursor": true, "scene": true, "ops": true,
}

// Collab — лёгкий броадкаст совместного редактирования: НИЧЕГО не сохраняет в
// БД, только публикует board_collab:<kind> в комнаты владельца и всех адресатов
// (включая отправителя — клиент отфильтрует по user_id). Доступ — владелец или
// адресат; kind=scene и kind=ops требуют права правки (владелец/can_edit).
//
// Горячий путь (cursor/scene идут на каждое действие): ФИО отправителя кладётся
// в payload ТОЛЬКО на join — клиент кэширует его по user_id; cursor/leave/scene
// поле fio не несут, лишний запрос в users на каждое событие не делается.
func (s *Service) Collab(ctx context.Context, userID, boardID int64, kind string, cursor *domain.CollabCursor, scene, ops json.RawMessage, title *string) error {
	if !collabKinds[kind] {
		return domain.ErrBadCollabKind
	}
	n, access, err := s.requireReadable(ctx, userID, boardID)
	if err != nil {
		return err
	}
	if (kind == "scene" || kind == "ops") && access == domain.AccessView {
		return domain.ErrMemberReadOnly
	}
	payload := map[string]any{"board_id": boardID, "user_id": userID}
	if kind == "join" {
		if u, err := s.users.GetUser(ctx, userID); err == nil && u != nil {
			payload["fio"] = u.FIO
		}
	}
	if cursor != nil {
		payload["cursor"] = cursor
	}
	if scene != nil {
		payload["scene"] = scene
	}
	// ops — пообъектные правки холста (upsert/remove): сервер их не разбирает,
	// а доставляет как есть — соавторы применяют адресно, не затирая сцену.
	if ops != nil {
		payload["ops"] = ops
	}
	// Название — часть live-правки (kind=scene, то же право): редактор шлёт его
	// вместе со сценой, чтобы у соавторов заголовок менялся в реальном
	// времени, а не только после PATCH-сохранения.
	if title != nil && kind == "scene" {
		payload["title"] = *title
	}
	s.bus.Publish(ctx, "board_collab:"+kind, s.boardRooms(ctx, boardID, n.OwnerID), payload)
	return nil
}
