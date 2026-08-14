package service

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/board/internal/domain"
)

// ListBoardsParams — параметры выборки плиток раздела.
type ListBoardsParams struct {
	FolderID  *int64
	FolderSet bool
	Search    string
	Archived  bool
}

// ListBoards — плитки владельца по фильтру; при просмотре ЧУЖОЙ (расшаренной)
// папки — её доски с owner/my_access.
func (s *Service) ListBoards(ctx context.Context, userID int64, p ListBoardsParams) ([]*domain.Board, error) {
	f := domain.BoardListFilter{
		OwnerID: userID, FolderID: p.FolderID, FolderSet: p.FolderSet,
		Search: strings.TrimSpace(p.Search), Archived: p.Archived,
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
			f.OwnerID = 0 // доски владельца папки
			f.Archived = false
			boards, err := s.repo.ListBoards(ctx, f)
			if err != nil {
				return nil, err
			}
			s.decorateShared(ctx, boards, fol.OwnerID, access)
			return boards, nil
		}
	}
	boards, err := s.repo.ListBoards(ctx, f)
	if err != nil {
		return nil, err
	}
	s.markSharedByMe(ctx, boards)
	if placed := s.recipientBoardsFor(ctx, userID, p); placed != nil {
		boards = append(boards, placed...)
	}
	return boards, nil
}

// recipientBoardsFor — чужие доски, размещённые мной в текущем scope проводника
// (папка/корень) или в личном архиве. Только для «своих» видов без поиска
// (поиск идёт своим путём — глобально).
func (s *Service) recipientBoardsFor(ctx context.Context, userID int64, p ListBoardsParams) []*domain.Board {
	if p.Search != "" {
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
	placed, err := s.repo.ListRecipientBoards(ctx, userID, s.companyIDs(ctx, userID), scope, folderID)
	if err != nil {
		s.log.Warn("boards.recipient_boards_failed", "user", userID, "error", err)
		return nil
	}
	return placed
}

// ListSharedBoards — чужие доски, доступные мне адресно или через расшаренную
// папку («поделились со мной»): плитки без сцены, с владельцем и my_access.
func (s *Service) ListSharedBoards(ctx context.Context, userID int64, search string) ([]*domain.Board, error) {
	return s.repo.ListSharedWithMe(ctx, userID, s.companyIDs(ctx, userID), strings.TrimSpace(search))
}

// decorateShared — проставить owner и my_access плиткам чужой расшаренной папки.
func (s *Service) decorateShared(ctx context.Context, boards []*domain.Board, ownerID int64, access string) {
	if len(boards) == 0 {
		return
	}
	var name string
	var avatar *string
	if owner, err := s.users.GetUser(ctx, ownerID); err == nil && owner != nil {
		name, avatar = owner.FIO, owner.AvatarPath
	}
	for _, n := range boards {
		n.OwnerName, n.OwnerAvatar, n.MyAccess = name, avatar, access
	}
}

// markSharedByMe — проставить SharedByMe плиткам владельца (значок «расшарено»).
func (s *Service) markSharedByMe(ctx context.Context, boards []*domain.Board) {
	if len(boards) == 0 {
		return
	}
	ids := make([]int64, len(boards))
	for i, n := range boards {
		ids[i] = n.ID
	}
	shared, err := s.repo.SharedByMeBoardIDs(ctx, ids)
	if err != nil {
		return
	}
	for _, n := range boards {
		if shared[n.ID] {
			n.SharedByMe = true
		}
	}
}

// GetBoard — полная доска (со сценой), доступная пользователю: своя или открытая
// шаром/папкой; my_access — owner | edit | view.
func (s *Service) GetBoard(ctx context.Context, userID, id int64) (*domain.Board, error) {
	n, access, err := s.requireReadable(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	n.MyAccess = access
	if access != domain.AccessOwner {
		if owner, err := s.users.GetUser(ctx, n.OwnerID); err == nil && owner != nil {
			n.OwnerName, n.OwnerAvatar = owner.FIO, owner.AvatarPath
		}
	}
	return n, nil
}

// CreateBoard — новая доска (опционально в папке владельца).
func (s *Service) CreateBoard(ctx context.Context, userID int64, title string, folderID *int64) (*domain.Board, error) {
	if err := s.checkOwnFolder(ctx, userID, folderID); err != nil {
		return nil, err
	}
	if err := s.ensureLimit(ctx, userID); err != nil {
		return nil, err
	}
	n := &domain.Board{OwnerID: userID, Title: title, Scene: domain.EmptyScene(), FolderID: folderID}
	if err := s.repo.CreateBoard(ctx, n); err != nil {
		return nil, err
	}
	s.publishBoard(ctx, "board:created", n)
	return n, nil
}

// UpdateBoard — частичная правка: nil-поля не меняются. При правке сцены сервер
// пересчитывает text_content (надписи и стикеры — для поиска). Color/Archived/
// Pinned — только владелец; Title/Scene — владелец, адресат с can_edit или
// edit-ссылка.
func (s *Service) UpdateBoard(ctx context.Context, userID, id int64, u domain.BoardUpdate) (*domain.Board, error) {
	if u.Color != nil && *u.Color != "" && !domain.BoardColors[*u.Color] {
		return nil, domain.ErrBadColor
	}
	n, access, err := s.requireReadable(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	// Адресату — личный архив «только у меня» (даже при доступе только на чтение):
	// это моя организация, а не правка содержимого владельца.
	if access != domain.AccessOwner && u.Archived != nil {
		if err := s.repo.SetBoardRecipientArchived(ctx, userID, id, *u.Archived); err != nil {
			return nil, err
		}
		resp := *n // личный архив — не трогаем доску владельца
		resp.Archived = *u.Archived
		return s.publishRecipientBoard(ctx, userID, &resp, access), nil
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
	case domain.AccessEdit: // адресату — только title/сцену
	default:
		return nil, domain.ErrMemberReadOnly
	}
	return s.applyUpdate(ctx, n, u.Title, u.Scene)
}

// applyUpdate — общая запись правки (владелец, адресат с can_edit, edit-ссылка).
func (s *Service) applyUpdate(ctx context.Context, n *domain.Board, title *string, scene json.RawMessage) (*domain.Board, error) {
	if title != nil {
		n.Title = *title
	}
	if scene != nil {
		if !json.Valid(scene) {
			return nil, domain.ErrBadScene
		}
		n.Scene = scene
		n.TextContent = domain.SceneText(scene)
	}
	if err := s.repo.UpdateBoard(ctx, n); err != nil {
		return nil, err
	}
	s.bus.Publish(ctx, "board:updated", s.boardRooms(ctx, n.ID, n.OwnerID), boardPayload(n))
	return n, nil
}

// DeleteBoard — удаление доски вместе с её картинками в хранилище.
func (s *Service) DeleteBoard(ctx context.Context, userID, id int64) error {
	n, err := s.requireOwned(ctx, userID, id)
	if err != nil {
		return err
	}
	rooms := s.boardRooms(ctx, id, userID) // до удаления — аудитория ещё цела
	keys := domain.SceneImageKeys(n.Scene)
	if n.PreviewPath != "" {
		keys = append(keys, n.PreviewPath)
	}
	if err := s.repo.DeleteBoard(ctx, id); err != nil {
		return err
	}
	if len(keys) > 0 {
		s.files.RemoveFor(ctx, userID, 0, keys)
	}
	s.bus.Publish(ctx, "board:deleted", rooms, map[string]any{"id": id, "owner_id": userID})
	return nil
}

// MoveBoard — разложить доску по папкам (folderID nil — в корень). Владелец
// двигает саму доску; адресат шаринга — раскладывает ЧУЖУЮ доску по СВОИМ
// папкам через личный оверлей (у владельца ничего не меняется). Цель — всегда
// моя папка (checkOwnFolder).
func (s *Service) MoveBoard(ctx context.Context, userID, id int64, folderID *int64) (*domain.Board, error) {
	n, access, err := s.requireReadable(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if err := s.checkOwnFolder(ctx, userID, folderID); err != nil {
		return nil, err
	}
	if access == domain.AccessOwner {
		if err := s.repo.MoveBoard(ctx, id, folderID); err != nil {
			return nil, err
		}
		n.FolderID = folderID
		s.publishBoard(ctx, "board:updated", n)
		return n, nil
	}
	if err := s.repo.SetBoardRecipientPlacement(ctx, userID, id, folderID); err != nil {
		return nil, err
	}
	resp := *n // не мутируем доску владельца — оверлей чисто мой
	resp.FolderID = folderID
	resp.Archived = false
	return s.publishRecipientBoard(ctx, userID, &resp, access), nil
}

// publishRecipientBoard — оформить чужую доску как размещённую мной (owner-поля,
// my_access, без личных тегов) и разослать событие только в мою комнату.
func (s *Service) publishRecipientBoard(ctx context.Context, userID int64, n *domain.Board, access string) *domain.Board {
	s.decorateShared(ctx, []*domain.Board{n}, n.OwnerID, access)
	s.bus.Publish(ctx, "board:updated", []string{userRoom(userID)}, boardPayload(n))
	return n
}

// CopyBoard — дубликат доски владельца (в той же папке).
func (s *Service) CopyBoard(ctx context.Context, userID, id int64) (*domain.Board, error) {
	src, err := s.requireOwned(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	cp := &domain.Board{
		OwnerID: userID, FolderID: src.FolderID, Title: copyTitle(src.Title),
		Color: src.Color, Scene: src.Scene, TextContent: src.TextContent,
	}
	if err := s.repo.CreateBoard(ctx, cp); err != nil {
		return nil, err
	}
	s.publishBoard(ctx, "board:created", cp)
	return cp, nil
}

/*
CheckUpload — можно ли класть на этот холст. Зовётся ДО первой части файла,

	пришедшего чанками: отказывать на сборке поздно.
*/
func (s *Service) CheckUpload(ctx context.Context, userID, boardID int64) error {
	_, access, err := s.requireReadable(ctx, userID, boardID)
	if err != nil {
		return err
	}
	if access != domain.AccessOwner && access != domain.AccessEdit {
		return domain.ErrMemberReadOnly
	}
	return nil
}

// UploadStream — картинка холста, собранная из частей.
func (s *Service) UploadStream(ctx context.Context, userID, boardID int64,
	fileName string, r io.Reader, size int64) (string, error) {

	if err := s.CheckUpload(ctx, userID, boardID); err != nil {
		return "", err
	}
	key, err := s.files.SaveStreamFor(ctx, userID, 0, fileName, r, size)
	if err != nil {
		return "", err
	}
	return "/uploads/" + key, nil
}

// Upload — картинка на холст: владелец или адресат с правом правки.
func (s *Service) Upload(ctx context.Context, userID, boardID int64, fileName string, data []byte) (string, error) {
	if err := s.CheckUpload(ctx, userID, boardID); err != nil {
		return "", err
	}
	key, err := s.files.SaveFor(ctx, userID, 0, fileName, data)
	if err != nil {
		return "", err
	}
	return "/uploads/" + key, nil
}

// SetPreview — миниатюра доски для плитки списка: растр снимает сам холст
// (canvas.toBlob), сервер только хранит ключ и подчищает предыдущий файл.
// Право — как на правку содержимого.
func (s *Service) SetPreview(ctx context.Context, userID, boardID int64, data []byte) (*domain.Board, error) {
	n, access, err := s.requireReadable(ctx, userID, boardID)
	if err != nil {
		return nil, err
	}
	if access != domain.AccessOwner && access != domain.AccessEdit {
		return nil, domain.ErrMemberReadOnly
	}
	key, err := s.files.SaveFor(ctx, userID, 0, "preview.png", data)
	if err != nil {
		return nil, err
	}
	old := n.PreviewPath
	if err := s.repo.SetBoardPreview(ctx, boardID, key); err != nil {
		s.files.RemoveFor(ctx, userID, 0, []string{key})
		return nil, err
	}
	if old != "" && old != key {
		s.files.RemoveFor(ctx, userID, 0, []string{old})
	}
	n.PreviewPath = key
	n.PreviewURL = "/uploads/" + key
	s.publishBoard(ctx, "board:updated", n)
	return n, nil
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
