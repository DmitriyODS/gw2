// Package service — бизнес-логика notesvc: личные заметки пользователя с
// rich-текстом (документ TipTap), группами-фильтрами, публичными ссылками
// (view/edit), адресным шарингом конкретным пользователям платформы (view/edit)
// и лёгким collab-броадкастом совместного редактирования. Скоуп — по владельцу
// (не по компании): заметка личная и кросс-компанийная. Сокет-события клиентам
// публикуются в Redis gw2:notes:events (доставляет gatewaysvc).
package service

import (
	"log/slog"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// excerptRunes — длина превью текста в плитке-стикере (синхронно с left(...)
// в repo-выборке списка).
const excerptRunes = 300

type Service struct {
	repo    domain.NoteRepository
	users   domain.UserReader
	files   domain.FileStore
	bus     domain.EventBus
	limiter domain.WriteLimiter
	log     *slog.Logger
}

type Deps struct {
	Repo    domain.NoteRepository
	Users   domain.UserReader
	Files   domain.FileStore
	Bus     domain.EventBus
	Limiter domain.WriteLimiter
	Log     *slog.Logger
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, users: d.Users, files: d.Files, bus: d.Bus, limiter: d.Limiter, log: d.Log}
}

// requireOwned — заметка во владении пользователя или доменная 404 (чужие
// заметки не раскрываются даже существованием).
func (s *Service) requireOwned(ctx domain.Ctx, userID, id int64) (*domain.Note, error) {
	n, err := s.repo.GetNote(ctx, id)
	if err != nil {
		return nil, err
	}
	if n == nil || n.OwnerID != userID {
		return nil, domain.ErrNoteNotFound
	}
	return n, nil
}

// requireReadable — заметка, доступная пользователю на чтение: своя
// (access=owner) или открытая адресно (edit|view). Чужая без доступа — единая
// 404, как и в requireOwned.
func (s *Service) requireReadable(ctx domain.Ctx, userID, id int64) (*domain.Note, string, error) {
	n, err := s.repo.GetNote(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if n == nil {
		return nil, "", domain.ErrNoteNotFound
	}
	if n.OwnerID == userID {
		return n, domain.AccessOwner, nil
	}
	found, canEdit, err := s.repo.GetMember(ctx, id, userID)
	if err != nil {
		return nil, "", err
	}
	if !found {
		return nil, "", domain.ErrNoteNotFound
	}
	if canEdit {
		return n, domain.AccessEdit, nil
	}
	return n, domain.AccessView, nil
}

// noteRooms — WS-комнаты доставки событий заметки: владелец + все адресаты.
// Так чужие клиенты получают изменения вживую, а посторонним события не утекают.
func (s *Service) noteRooms(ctx domain.Ctx, noteID, ownerID int64) []string {
	rooms := []string{userRoom(ownerID)}
	ids, err := s.repo.MemberIDs(ctx, noteID)
	if err != nil {
		s.log.Warn("notes.member_ids_failed", "note", noteID, "error", err)
		return rooms
	}
	for _, id := range ids {
		rooms = append(rooms, userRoom(id))
	}
	return rooms
}

// requireGroupOwned — группа во владении пользователя или доменная 404.
func (s *Service) requireGroupOwned(ctx domain.Ctx, userID, id int64) (*domain.Group, error) {
	g, err := s.repo.GetGroup(ctx, id)
	if err != nil {
		return nil, err
	}
	if g == nil || g.OwnerID != userID {
		return nil, domain.ErrGroupNotFound
	}
	return g, nil
}

func excerptOf(text string) string {
	r := []rune(text)
	if len(r) > excerptRunes {
		r = r[:excerptRunes]
	}
	return string(r)
}

func userRoom(id int64) string { return "user_" + strconv.FormatInt(id, 10) }

// notePayload — событие плитки (без doc: полный документ клиент тянет REST'ом
// при открытии страницы).
func notePayload(n *domain.Note) map[string]any {
	groups := n.GroupIDs
	if groups == nil {
		groups = []int64{}
	}
	p := map[string]any{
		"id": n.ID, "owner_id": n.OwnerID, "title": n.Title, "color": n.Color,
		"archived": n.Archived, "pinned_at": n.PinnedAt,
		"excerpt": excerptOf(n.TextContent), "group_ids": groups,
		"created_at": n.CreatedAt, "updated_at": n.UpdatedAt,
	}
	if n.OwnerName != "" {
		p["owner_name"] = n.OwnerName
		p["owner_avatar"] = n.OwnerAvatar
	}
	return p
}

func (s *Service) publishNote(ctx domain.Ctx, event string, n *domain.Note) {
	s.bus.Publish(ctx, event, []string{userRoom(n.OwnerID)}, notePayload(n))
}
