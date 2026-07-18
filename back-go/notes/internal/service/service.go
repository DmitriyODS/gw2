// Package service — бизнес-логика notesvc: личные заметки пользователя с
// rich-текстом (документ TipTap), иерархическими папками, тегами-метками,
// публичными ссылками (view/edit) и адресным шарингом заметок И папок —
// конкретным пользователям платформы либо целым компаниям (view/edit), с лёгким
// collab-броадкастом совместного редактирования. Заметка/папка принадлежит
// одному пользователю и кросс-компанийна; доступ другим — по шарам (эффективный
// доступ учитывает расшаренные папки-предки). Сокет-события клиентам публикуются
// в Redis gw2:notes:events (доставляет gatewaysvc).
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
	repo     domain.NoteRepository
	users    domain.UserReader
	files    domain.FileStore
	bus      domain.EventBus
	limiter  domain.WriteLimiter
	embedder domain.Embedder
	log      *slog.Logger
}

type Deps struct {
	Repo     domain.NoteRepository
	Users    domain.UserReader
	Files    domain.FileStore
	Bus      domain.EventBus
	Limiter  domain.WriteLimiter
	Embedder domain.Embedder // nil — ИИ-поиск выключен (фолбэк на текстовый)
	Log      *slog.Logger
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, users: d.Users, files: d.Files, bus: d.Bus,
		limiter: d.Limiter, embedder: d.Embedder, log: d.Log}
}

// aiEnabled — ИИ-поиск доступен (клиент aisvc настроен).
func (s *Service) aiEnabled() bool { return s.embedder != nil && s.embedder.Enabled() }

// companyIDs — компании пользователя (скоуп «расшарено моей компании»). Ошибка
// чтения не должна ронять раздел — возвращаем пустой скоуп (доступ только личный).
func (s *Service) companyIDs(ctx domain.Ctx, userID int64) []int64 {
	ids, err := s.users.CompanyIDs(ctx, userID)
	if err != nil {
		s.log.Warn("notes.company_ids_failed", "user", userID, "error", err)
		return []int64{}
	}
	return ids
}

// requireOwned — заметка во владении пользователя или доменная 404.
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

// requireReadable — заметка, доступная пользователю на чтение: своя (owner) или
// открытая шаром / расшаренной папкой-предком (edit|view). Иначе — единая 404.
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
	found, canEdit, err := s.repo.NoteAccess(ctx, userID, s.companyIDs(ctx, userID), id, n.FolderID)
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

// requireTagOwned — тег во владении пользователя или доменная 404.
func (s *Service) requireTagOwned(ctx domain.Ctx, userID, id int64) (*domain.Tag, error) {
	t, err := s.repo.GetTag(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil || t.OwnerID != userID {
		return nil, domain.ErrTagNotFound
	}
	return t, nil
}

// noteRooms — WS-комнаты доставки событий заметки: владелец + вся аудитория
// (адресаты и участники компаний, включая доступ через расшаренные папки-предки).
func (s *Service) noteRooms(ctx domain.Ctx, noteID, ownerID int64) []string {
	return s.roomsFor(ownerID, s.audience(ctx, noteID, true))
}

// folderRooms — то же для папки.
func (s *Service) folderRooms(ctx domain.Ctx, folderID, ownerID int64) []string {
	return s.roomsFor(ownerID, s.audience(ctx, folderID, false))
}

func (s *Service) audience(ctx domain.Ctx, id int64, note bool) []int64 {
	var (
		ids []int64
		err error
	)
	if note {
		ids, err = s.repo.NoteAudienceUserIDs(ctx, id)
	} else {
		ids, err = s.repo.FolderAudienceUserIDs(ctx, id)
	}
	if err != nil {
		s.log.Warn("notes.audience_failed", "id", id, "note", note, "error", err)
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

// notePayload — событие плитки (без doc: полный документ клиент тянет REST'ом).
func notePayload(n *domain.Note) map[string]any {
	tags := n.TagIDs
	if tags == nil {
		tags = []int64{}
	}
	p := map[string]any{
		"id": n.ID, "owner_id": n.OwnerID, "title": n.Title, "color": n.Color,
		"archived": n.Archived, "folder_id": n.FolderID, "pinned_at": n.PinnedAt,
		"excerpt": excerptOf(n.TextContent), "tag_ids": tags,
		"shared_by_me": n.SharedByMe,
		"created_at":   n.CreatedAt, "updated_at": n.UpdatedAt,
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

func (s *Service) publishNote(ctx domain.Ctx, event string, n *domain.Note) {
	s.bus.Publish(ctx, event, s.noteRooms(ctx, n.ID, n.OwnerID), notePayload(n))
}
