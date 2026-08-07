// Package service — бизнес-логика boardsvc: личные доски пользователя со
// сценой рисования (объекты холста), иерархическими папками, тегами-метками,
// публичными ссылками (view/edit) и адресным шарингом досок И папок —
// конкретным пользователям платформы либо целым компаниям (view/edit), с лёгким
// collab-броадкастом совместного редактирования. Доска/папка принадлежит
// одному пользователю и кросс-компанийна; доступ другим — по шарам (эффективный
// доступ учитывает расшаренные папки-предки). Сокет-события клиентам публикуются
// в Redis gw2:board:events (доставляет gatewaysvc).
package service

import (
	"log/slog"
	"strconv"

	"github.com/DmitriyODS/gw2/back-go/board/internal/domain"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
)

// excerptRunes — длина превью текста в плитке-стикере (синхронно с left(...)
// в repo-выборке списка).
const excerptRunes = 300

type Service struct {
	repo    domain.BoardRepository
	users   domain.UserReader
	files   domain.FileStore
	bus     domain.EventBus
	limiter domain.WriteLimiter
	log     *slog.Logger
	// billing — лимиты тарифа (WithBilling; nil — ограничений нет).
	billing *billingclient.Client
}

type Deps struct {
	Repo    domain.BoardRepository
	Users   domain.UserReader
	Files   domain.FileStore
	Bus     domain.EventBus
	Limiter domain.WriteLimiter
	Log     *slog.Logger
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, users: d.Users, files: d.Files, bus: d.Bus,
		limiter: d.Limiter, log: d.Log}
}

// companyIDs — компании пользователя (скоуп «расшарено моей компании»). Ошибка
// чтения не должна ронять раздел — возвращаем пустой скоуп (доступ только личный).
func (s *Service) companyIDs(ctx domain.Ctx, userID int64) []int64 {
	ids, err := s.users.CompanyIDs(ctx, userID)
	if err != nil {
		s.log.Warn("boards.company_ids_failed", "user", userID, "error", err)
		return []int64{}
	}
	return ids
}

// requireOwned — доска во владении пользователя или доменная 404.
func (s *Service) requireOwned(ctx domain.Ctx, userID, id int64) (*domain.Board, error) {
	n, err := s.repo.GetBoard(ctx, id)
	if err != nil {
		return nil, err
	}
	if n == nil || n.OwnerID != userID {
		return nil, domain.ErrBoardNotFound
	}
	return n, nil
}

// requireReadable — доска, доступная пользователю на чтение: своя (owner) или
// открытая шаром / расшаренной папкой-предком (edit|view). Иначе — единая 404.
func (s *Service) requireReadable(ctx domain.Ctx, userID, id int64) (*domain.Board, string, error) {
	n, err := s.repo.GetBoard(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if n == nil {
		return nil, "", domain.ErrBoardNotFound
	}
	if n.OwnerID == userID {
		return n, domain.AccessOwner, nil
	}
	found, canEdit, err := s.repo.BoardAccess(ctx, userID, s.companyIDs(ctx, userID), id, n.FolderID)
	if err != nil {
		return nil, "", err
	}
	if !found {
		return nil, "", domain.ErrBoardNotFound
	}
	if canEdit {
		return n, domain.AccessEdit, nil
	}
	return n, domain.AccessView, nil
}

// requireFolderOwned — папка во владении пользователя или доменная 404.
func (s *Service) requireFolderOwned(ctx domain.Ctx, userID, id int64) (*domain.Folder, error) {
	f, err := s.repo.GetFolder(ctx, id)
	if err != nil {
		return nil, err
	}
	if f == nil || f.OwnerID != userID {
		return nil, domain.ErrFolderNotFound
	}
	return f, nil
}

// requireFolderReadable — папка, доступная пользователю: своя (owner) или
// расшаренная ей/предку (edit|view).
func (s *Service) requireFolderReadable(ctx domain.Ctx, userID, id int64) (*domain.Folder, string, error) {
	f, err := s.repo.GetFolder(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if f == nil {
		return nil, "", domain.ErrFolderNotFound
	}
	if f.OwnerID == userID {
		return f, domain.AccessOwner, nil
	}
	found, canEdit, err := s.repo.FolderAccess(ctx, userID, s.companyIDs(ctx, userID), id)
	if err != nil {
		return nil, "", err
	}
	if !found {
		return nil, "", domain.ErrFolderNotFound
	}
	if canEdit {
		return f, domain.AccessEdit, nil
	}
	return f, domain.AccessView, nil
}

// boardRooms — WS-комнаты доставки событий доски: владелец + вся аудитория
// (адресаты и участники компаний, включая доступ через расшаренные папки-предки).
func (s *Service) boardRooms(ctx domain.Ctx, boardID, ownerID int64) []string {
	return s.roomsFor(ownerID, s.audience(ctx, boardID, true))
}

// folderRooms — то же для папки.
func (s *Service) folderRooms(ctx domain.Ctx, folderID, ownerID int64) []string {
	return s.roomsFor(ownerID, s.audience(ctx, folderID, false))
}

func (s *Service) audience(ctx domain.Ctx, id int64, board bool) []int64 {
	var (
		ids []int64
		err error
	)
	if board {
		ids, err = s.repo.BoardAudienceUserIDs(ctx, id)
	} else {
		ids, err = s.repo.FolderAudienceUserIDs(ctx, id)
	}
	if err != nil {
		s.log.Warn("boards.audience_failed", "id", id, "board", board, "error", err)
		return nil
	}
	return ids
}

func (s *Service) roomsFor(ownerID int64, audience []int64) []string {
	rooms := []string{userRoom(ownerID)}
	for _, id := range audience {
		if id != ownerID {
			rooms = append(rooms, userRoom(id))
		}
	}
	return rooms
}

func excerptOf(text string) string {
	r := []rune(text)
	if len(r) > excerptRunes {
		r = r[:excerptRunes]
	}
	return string(r)
}

func userRoom(id int64) string { return "user_" + strconv.FormatInt(id, 10) }

// boardPayload — событие плитки (без сцены: холст клиент тянет REST'ом).
func boardPayload(n *domain.Board) map[string]any {
	p := map[string]any{
		"id": n.ID, "owner_id": n.OwnerID, "title": n.Title, "color": n.Color,
		"archived": n.Archived, "folder_id": n.FolderID, "pinned_at": n.PinnedAt,
		"excerpt":      excerptOf(n.TextContent),
		"shared_by_me": n.SharedByMe, "preview_url": n.PreviewURL,
		"created_at": n.CreatedAt, "updated_at": n.UpdatedAt,
	}
	if n.OwnerName != "" {
		p["owner_name"] = n.OwnerName
		p["owner_avatar"] = n.OwnerAvatar
	}
	if n.MyAccess != "" {
		p["my_access"] = n.MyAccess
	}
	return p
}

func (s *Service) publishBoard(ctx domain.Ctx, event string, n *domain.Board) {
	s.bus.Publish(ctx, event, s.boardRooms(ctx, n.ID, n.OwnerID), boardPayload(n))
}
