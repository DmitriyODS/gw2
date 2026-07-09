// Package service — бизнес-логика notesvc: личные заметки пользователя с
// rich-текстом (документ TipTap), группами-фильтрами и публичными ссылками
// (view/edit). Скоуп — по владельцу (не по компании): заметка личная и
// кросс-компанийная. Сокет-события клиентам публикуются в Redis
// gw2:notes:events (доставляет gatewaysvc).
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
	files   domain.FileStore
	bus     domain.EventBus
	limiter domain.WriteLimiter
	log     *slog.Logger
}

type Deps struct {
	Repo    domain.NoteRepository
	Files   domain.FileStore
	Bus     domain.EventBus
	Limiter domain.WriteLimiter
	Log     *slog.Logger
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, files: d.Files, bus: d.Bus, limiter: d.Limiter, log: d.Log}
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
	return map[string]any{
		"id": n.ID, "owner_id": n.OwnerID, "title": n.Title, "color": n.Color,
		"excerpt": excerptOf(n.TextContent), "group_ids": groups,
		"created_at": n.CreatedAt, "updated_at": n.UpdatedAt,
	}
}

func (s *Service) publishNote(ctx domain.Ctx, event string, n *domain.Note) {
	s.bus.Publish(ctx, event, []string{userRoom(n.OwnerID)}, notePayload(n))
}
