package service

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"slices"
	"sort"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/board/internal/domain"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// ── Fakes ────────────────────────────────────────────────────────────

type fakeRepo struct {
	boards      map[int64]*domain.Board
	folders     map[int64]*domain.Folder
	shares      map[string]*domain.Share
	boardUsers  map[int64]map[int64]bool // boardID → userID → canEdit
	boardCos    map[int64]map[int64]bool // boardID → companyID → canEdit
	folderUsers map[int64]map[int64]bool
	folderCos   map[int64]map[int64]bool
	boardState  map[int64]map[int64]*recipBoardState  // userID → boardID → оверлей
	folderState map[int64]map[int64]*recipFolderState // userID → folderID → оверлей
	next        int64
}

type recipBoardState struct {
	folderID *int64
	archived bool
}
type recipFolderState struct {
	parentID *int64
	archived bool
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		boards: map[int64]*domain.Board{}, folders: map[int64]*domain.Folder{},
		shares:     map[string]*domain.Share{},
		boardUsers: map[int64]map[int64]bool{}, boardCos: map[int64]map[int64]bool{},
		folderUsers: map[int64]map[int64]bool{}, folderCos: map[int64]map[int64]bool{},
		boardState: map[int64]map[int64]*recipBoardState{}, folderState: map[int64]map[int64]*recipFolderState{},
	}
}

func (f *fakeRepo) id() int64 { f.next++; return f.next }

// ancestors — id папки и всех её предков.
func (f *fakeRepo) ancestors(folderID *int64) []int64 {
	out := []int64{}
	for folderID != nil {
		fol := f.folders[*folderID]
		if fol == nil {
			break
		}
		out = append(out, fol.ID)
		folderID = fol.ParentID
	}
	return out
}

// ── Доски ──
func (f *fakeRepo) ListBoards(_ domain.Ctx, fl domain.BoardListFilter) ([]*domain.Board, error) {
	out := []*domain.Board{}
	for _, n := range f.boards {
		if fl.OwnerID > 0 && n.OwnerID != fl.OwnerID {
			continue
		}
		if n.Archived != fl.Archived {
			continue
		}
		if fl.FolderSet {
			var fid int64
			if n.FolderID != nil {
				fid = *n.FolderID
			}
			var want int64
			if fl.FolderID != nil {
				want = *fl.FolderID
			}
			if fid != want {
				continue
			}
		}
		out = append(out, n)
	}
	slices.SortFunc(out, func(a, b *domain.Board) int { return int(b.ID - a.ID) })
	return out, nil
}
func (f *fakeRepo) SharedByMeBoardIDs(_ domain.Ctx, ids []int64) (map[int64]bool, error) {
	res := map[int64]bool{}
	for _, id := range ids {
		if len(f.boardUsers[id]) > 0 || len(f.boardCos[id]) > 0 {
			res[id] = true
		}
	}
	return res, nil
}

func (f *fakeRepo) ListSharedWithMe(_ domain.Ctx, userID int64, companyIDs []int64, _ string) ([]*domain.Board, error) {
	out := []*domain.Board{}
	for _, n := range f.boards {
		if n.OwnerID == userID {
			continue
		}
		if f.boardState[userID][n.ID] != nil { // размещена мной / в личном архиве
			continue
		}
		found, _, _ := f.BoardAccess(nil, userID, companyIDs, n.ID, n.FolderID)
		if found {
			out = append(out, n)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetBoard(_ domain.Ctx, id int64) (*domain.Board, error) {
	return f.boards[id], nil
}
func (f *fakeRepo) CreateBoard(_ domain.Ctx, n *domain.Board) error {
	n.ID = f.id()
	f.boards[n.ID] = n
	return nil
}
func (f *fakeRepo) UpdateBoard(_ domain.Ctx, n *domain.Board) error { f.boards[n.ID] = n; return nil }

// Раздел «Хранилище»: картинки живут внутри сцены, поэтому и список, и
// вырезание идут по ней.
func (f *fakeRepo) BoardScenesOf(_ domain.Ctx, ownerID int64) ([]*domain.Board, error) {
	out := []*domain.Board{}
	for _, b := range f.boards {
		if b.OwnerID == ownerID {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (f *fakeRepo) UpdateBoardScene(_ domain.Ctx, id int64, scene json.RawMessage, text string) error {
	if b := f.boards[id]; b != nil {
		b.Scene, b.TextContent = scene, text
	}
	return nil
}

func (f *fakeRepo) DeleteBoard(_ domain.Ctx, id int64) error        { delete(f.boards, id); return nil }
func (f *fakeRepo) MoveBoard(_ domain.Ctx, id int64, folderID *int64) error {
	if n := f.boards[id]; n != nil {
		n.FolderID = folderID
	}
	return nil
}
func (f *fakeRepo) SetBoardRecipientPlacement(_ domain.Ctx, userID, boardID int64, folderID *int64) error {
	if f.boardState[userID] == nil {
		f.boardState[userID] = map[int64]*recipBoardState{}
	}
	f.boardState[userID][boardID] = &recipBoardState{folderID: folderID}
	return nil
}

func (f *fakeRepo) SetBoardRecipientArchived(_ domain.Ctx, userID, boardID int64, archived bool) error {
	if f.boardState[userID] == nil {
		f.boardState[userID] = map[int64]*recipBoardState{}
	}
	st := f.boardState[userID][boardID]
	if st == nil {
		st = &recipBoardState{}
		f.boardState[userID][boardID] = st
	}
	st.archived = archived
	return nil
}

func (f *fakeRepo) ListRecipientBoards(_ domain.Ctx, userID int64, companyIDs []int64, scope domain.RecipientScope, folderID *int64) ([]*domain.Board, error) {
	out := []*domain.Board{}
	for boardID, st := range f.boardState[userID] {
		n := f.boards[boardID]
		if n == nil || n.OwnerID == userID {
			continue
		}
		found, canEdit, _ := f.BoardAccess(nil, userID, companyIDs, n.ID, n.FolderID)
		if !found {
			continue
		}
		match := false
		switch scope {
		case domain.RecipientArchive:
			match = st.archived
		case domain.RecipientRoot:
			match = st.folderID == nil && !st.archived
		default:
			match = !st.archived && st.folderID != nil && folderID != nil && *st.folderID == *folderID
		}
		if !match {
			continue
		}
		cp := *n
		cp.FolderID, cp.Archived = st.folderID, st.archived
		cp.MyAccess = domain.AccessView
		if canEdit {
			cp.MyAccess = domain.AccessEdit
		}
		out = append(out, &cp)
	}
	return out, nil
}

func (f *fakeRepo) SetFolderRecipientPlacement(_ domain.Ctx, userID, folderID int64, parentID *int64) error {
	if f.folderState[userID] == nil {
		f.folderState[userID] = map[int64]*recipFolderState{}
	}
	f.folderState[userID][folderID] = &recipFolderState{parentID: parentID}
	return nil
}

func (f *fakeRepo) ListRecipientFolders(_ domain.Ctx, userID int64, companyIDs []int64) ([]*domain.Folder, error) {
	out := []*domain.Folder{}
	for fid, st := range f.folderState[userID] {
		if st.archived {
			continue
		}
		fol := f.folders[fid]
		if fol == nil || fol.OwnerID == userID {
			continue
		}
		found, canEdit, _ := f.FolderAccess(nil, userID, companyIDs, fid)
		if !found {
			continue
		}
		cp := *fol
		cp.ParentID = st.parentID
		cp.MyAccess = domain.AccessView
		if canEdit {
			cp.MyAccess = domain.AccessEdit
		}
		out = append(out, &cp)
	}
	return out, nil
}

// ── Папки ──
func (f *fakeRepo) ListFolders(_ domain.Ctx, ownerID int64) ([]*domain.Folder, error) {
	out := []*domain.Folder{}
	for _, fol := range f.folders {
		if fol.OwnerID == ownerID {
			out = append(out, fol)
		}
	}
	return out, nil
}
func (f *fakeRepo) ListChildFolders(_ domain.Ctx, parentID int64) ([]*domain.Folder, error) {
	out := []*domain.Folder{}
	for _, fol := range f.folders {
		if fol.ParentID != nil && *fol.ParentID == parentID {
			out = append(out, fol)
		}
	}
	return out, nil
}
func (f *fakeRepo) ListSharedRootFolders(_ domain.Ctx, userID int64, companyIDs []int64) ([]*domain.Folder, error) {
	out := []*domain.Folder{}
	for _, fol := range f.folders {
		if fol.OwnerID == userID {
			continue
		}
		if f.folderState[userID][fol.ID] != nil { // размещена мной в своём дереве
			continue
		}
		if f.folderUsers[fol.ID][userID] {
			out = append(out, fol)
			continue
		}
		for _, cid := range companyIDs {
			if _, ok := f.folderCos[fol.ID][cid]; ok {
				out = append(out, fol)
				break
			}
		}
	}
	return out, nil
}
func (f *fakeRepo) GetFolder(_ domain.Ctx, id int64) (*domain.Folder, error) {
	return f.folders[id], nil
}
func (f *fakeRepo) CreateFolder(_ domain.Ctx, fol *domain.Folder) error {
	fol.ID = f.id()
	f.folders[fol.ID] = fol
	return nil
}
func (f *fakeRepo) UpdateFolder(_ domain.Ctx, id int64, name, color string) error {
	if fol := f.folders[id]; fol != nil {
		fol.Name, fol.Color = name, color
	}
	return nil
}
func (f *fakeRepo) MoveFolder(_ domain.Ctx, id int64, parentID *int64) error {
	if fol := f.folders[id]; fol != nil {
		fol.ParentID = parentID
	}
	return nil
}
func (f *fakeRepo) DeleteFolder(_ domain.Ctx, id int64) error                       { delete(f.folders, id); return nil }
func (f *fakeRepo) NextFolderPosition(_ domain.Ctx, _ int64, _ *int64) (int, error) { return 0, nil }
func (f *fakeRepo) IsDescendant(_ domain.Ctx, folderID, maybeAncestor int64) (bool, error) {
	cur := &folderID
	for cur != nil {
		if *cur == maybeAncestor {
			return true, nil
		}
		fol := f.folders[*cur]
		if fol == nil {
			break
		}
		cur = fol.ParentID
	}
	return false, nil
}
func (f *fakeRepo) ReparentChildren(_ domain.Ctx, folderID int64, newParent *int64) error {
	for _, fol := range f.folders {
		if fol.ParentID != nil && *fol.ParentID == folderID {
			fol.ParentID = newParent
		}
	}
	for _, n := range f.boards {
		if n.FolderID != nil && *n.FolderID == folderID {
			n.FolderID = newParent
		}
	}
	return nil
}
func (f *fakeRepo) CopyFolderTree(_ domain.Ctx, ownerID, folderID int64, newParent *int64) (int64, error) {
	src := f.folders[folderID]
	cp := &domain.Folder{OwnerID: ownerID, ParentID: newParent, Name: src.Name, Color: src.Color}
	cp.ID = f.id()
	f.folders[cp.ID] = cp
	return cp.ID, nil
}

// ── Теги ──
// ── Публичные ссылки ──
func (f *fakeRepo) ListShares(_ domain.Ctx, boardID int64) ([]*domain.Share, error) {
	out := []*domain.Share{}
	for _, s := range f.shares {
		if s.BoardID == boardID {
			out = append(out, s)
		}
	}
	return out, nil
}
func (f *fakeRepo) CreateShare(_ domain.Ctx, s *domain.Share) error {
	s.ID = f.id()
	f.shares[s.Code] = s
	return nil
}
func (f *fakeRepo) GetShareByCode(_ domain.Ctx, code string) (*domain.Share, error) {
	return f.shares[code], nil
}
func (f *fakeRepo) DeleteShare(_ domain.Ctx, id, boardID int64) error {
	for k, s := range f.shares {
		if s.ID == id && s.BoardID == boardID {
			delete(f.shares, k)
		}
	}
	return nil
}

// ── Шаринг досок ──
func (f *fakeRepo) ListBoardMembers(_ domain.Ctx, _ int64) ([]*domain.Member, error) { return nil, nil }
func (f *fakeRepo) UpsertBoardUserShare(_ domain.Ctx, boardID, userID int64, canEdit bool) error {
	if f.boardUsers[boardID] == nil {
		f.boardUsers[boardID] = map[int64]bool{}
	}
	f.boardUsers[boardID][userID] = canEdit
	return nil
}
func (f *fakeRepo) DeleteBoardUserShare(_ domain.Ctx, boardID, userID int64) error {
	delete(f.boardUsers[boardID], userID)
	return nil
}
func (f *fakeRepo) UpsertBoardCompanyShare(_ domain.Ctx, boardID, companyID int64, _ string, canEdit bool, _ int64) error {
	if f.boardCos[boardID] == nil {
		f.boardCos[boardID] = map[int64]bool{}
	}
	f.boardCos[boardID][companyID] = canEdit
	return nil
}
func (f *fakeRepo) DeleteBoardCompanyShare(_ domain.Ctx, boardID, companyID int64) error {
	delete(f.boardCos[boardID], companyID)
	return nil
}

// ── Шаринг папок ──
func (f *fakeRepo) ListFolderMembers(_ domain.Ctx, _ int64) ([]*domain.Member, error) {
	return nil, nil
}
func (f *fakeRepo) UpsertFolderUserShare(_ domain.Ctx, folderID, userID int64, canEdit bool) error {
	if f.folderUsers[folderID] == nil {
		f.folderUsers[folderID] = map[int64]bool{}
	}
	f.folderUsers[folderID][userID] = canEdit
	return nil
}
func (f *fakeRepo) DeleteFolderUserShare(_ domain.Ctx, folderID, userID int64) error {
	delete(f.folderUsers[folderID], userID)
	return nil
}
func (f *fakeRepo) UpsertFolderCompanyShare(_ domain.Ctx, folderID, companyID int64, _ string, canEdit bool, _ int64) error {
	if f.folderCos[folderID] == nil {
		f.folderCos[folderID] = map[int64]bool{}
	}
	f.folderCos[folderID][companyID] = canEdit
	return nil
}
func (f *fakeRepo) DeleteFolderCompanyShare(_ domain.Ctx, folderID, companyID int64) error {
	delete(f.folderCos[folderID], companyID)
	return nil
}

// ── Аудитория / доступ ──
func (f *fakeRepo) BoardAudienceUserIDs(_ domain.Ctx, _ int64) ([]int64, error)  { return nil, nil }
func (f *fakeRepo) FolderAudienceUserIDs(_ domain.Ctx, _ int64) ([]int64, error) { return nil, nil }

func (f *fakeRepo) BoardAccess(_ domain.Ctx, userID int64, companyIDs []int64, boardID int64, folderID *int64) (bool, bool, error) {
	found, canEdit := false, false
	mark := func(ok, ce bool) {
		if ok {
			found = true
			canEdit = canEdit || ce
		}
	}
	if ce, ok := f.boardUsers[boardID][userID]; ok {
		mark(true, ce)
	}
	for _, cid := range companyIDs {
		if ce, ok := f.boardCos[boardID][cid]; ok {
			mark(true, ce)
		}
	}
	for _, aid := range f.ancestors(folderID) {
		if ce, ok := f.folderUsers[aid][userID]; ok {
			mark(true, ce)
		}
		for _, cid := range companyIDs {
			if ce, ok := f.folderCos[aid][cid]; ok {
				mark(true, ce)
			}
		}
	}
	return found, canEdit, nil
}
func (f *fakeRepo) FolderAccess(_ domain.Ctx, userID int64, companyIDs []int64, folderID int64) (bool, bool, error) {
	fid := folderID
	return f.BoardAccess(nil, userID, companyIDs, 0, &fid)
}

func (f *fakeRepo) SetBoardPreview(_ domain.Ctx, boardID int64, key string) error {
	if n := f.boards[boardID]; n != nil {
		n.PreviewPath = key
	}
	return nil
}

// UserReader
type fakeUsers struct {
	users   map[int64]*domain.User
	members map[int64][]int64 // userID → companyIDs
	compNm  map[int64]string
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{users: map[int64]*domain.User{}, members: map[int64][]int64{}, compNm: map[int64]string{}}
}
func (u *fakeUsers) GetUser(_ domain.Ctx, id int64) (*domain.User, error) { return u.users[id], nil }
func (u *fakeUsers) UserCompanies(_ domain.Ctx, userID int64) ([]*domain.Company, error) {
	out := []*domain.Company{}
	for _, cid := range u.members[userID] {
		out = append(out, &domain.Company{ID: cid, Name: u.compNm[cid]})
	}
	return out, nil
}
func (u *fakeUsers) CompanyIDs(_ domain.Ctx, userID int64) ([]int64, error) {
	return u.members[userID], nil
}
func (u *fakeUsers) IsCompanyMember(_ domain.Ctx, userID, companyID int64) (bool, string, error) {
	if slices.Contains(u.members[userID], companyID) {
		return true, u.compNm[companyID], nil
	}
	return false, "", nil
}

type nopBus struct{}

func (nopBus) Publish(_ domain.Ctx, _ string, _ []string, _ any) {}

type nopFiles struct{}

func (nopFiles) SaveFor(_ context.Context, _, _ int64, _ string, _ []byte) (string, error) {
	return "boards/x", nil
}
func (nopFiles) RemoveFor(_ context.Context, _, _ int64, _ []string) {}
func (nopFiles) Remove(_ []string)                                   {}
func (nopFiles) Open(_ string) ([]byte, error)                       { return nil, nil }

type allowLimiter struct{}

func (allowLimiter) Allow(_ domain.Ctx, _ string) bool { return true }

func newSvc(repo *fakeRepo, users *fakeUsers) *Service {
	return New(Deps{Repo: repo, Users: users, Files: nopFiles{}, Bus: nopBus{}, Limiter: allowLimiter{}, Log: discardLogger()})
}

func ctx() context.Context { return context.Background() }

// ── Тесты ────────────────────────────────────────────────────────────

func TestBadColorRejected(t *testing.T) {
	repo, users := newFakeRepo(), newFakeUsers()
	s := newSvc(repo, users)
	b, err := s.CreateBoard(ctx(), 1, "Доска", nil)
	if err != nil {
		t.Fatalf("создание доски: %v", err)
	}
	bad := "chartreuse"
	if _, err := s.UpdateBoard(ctx(), 1, b.ID, domain.BoardUpdate{Color: &bad}); err != domain.ErrBadColor {
		t.Fatalf("неизвестный цвет должен отклоняться, получено %v", err)
	}
}

func TestFolderMoveCyclePrevented(t *testing.T) {
	repo, users := newFakeRepo(), newFakeUsers()
	s := newSvc(repo, users)
	parent, _ := s.CreateFolder(ctx(), 1, "p", "", nil)
	child, _ := s.CreateFolder(ctx(), 1, "c", "", &parent.ID)
	// Переместить родителя внутрь ребёнка — цикл.
	if _, err := s.MoveFolder(ctx(), 1, parent.ID, &child.ID); err != domain.ErrFolderCycle {
		t.Fatalf("want ErrFolderCycle, got %v", err)
	}
}

func TestDeleteFolderReparentsChildren(t *testing.T) {
	repo, users := newFakeRepo(), newFakeUsers()
	users.users[1] = &domain.User{ID: 1, IsActive: true}
	s := newSvc(repo, users)
	parent, _ := s.CreateFolder(ctx(), 1, "p", "", nil)
	mid, _ := s.CreateFolder(ctx(), 1, "m", "", &parent.ID)
	n, _ := s.CreateBoard(ctx(), 1, "n", &mid.ID)
	if err := s.DeleteFolder(ctx(), 1, mid.ID); err != nil {
		t.Fatal(err)
	}
	if repo.boards[n.ID].FolderID == nil || *repo.boards[n.ID].FolderID != parent.ID {
		t.Fatalf("board not reparented: %v", repo.boards[n.ID].FolderID)
	}
}

func TestShareCompanyRequiresMembership(t *testing.T) {
	repo, users := newFakeRepo(), newFakeUsers()
	users.users[1] = &domain.User{ID: 1, IsActive: true}
	users.members[1] = []int64{10}
	users.compNm[10] = "Acme"
	s := newSvc(repo, users)
	n, _ := s.CreateBoard(ctx(), 1, "n", nil)

	// Не член компании 99 → отказ.
	if _, err := s.ShareBoard(ctx(), 1, n.ID, domain.TargetCompany, 99, false); err != domain.ErrNotCompanyMember {
		t.Fatalf("want ErrNotCompanyMember, got %v", err)
	}
	// Член компании 10 → успех.
	if _, err := s.ShareBoard(ctx(), 1, n.ID, domain.TargetCompany, 10, true); err != nil {
		t.Fatalf("share company failed: %v", err)
	}
}

func TestAccessViaFolderShare(t *testing.T) {
	repo, users := newFakeRepo(), newFakeUsers()
	users.users[1] = &domain.User{ID: 1, IsActive: true} // владелец
	users.users[2] = &domain.User{ID: 2, IsActive: true} // адресат
	s := newSvc(repo, users)

	root, _ := s.CreateFolder(ctx(), 1, "root", "", nil)
	sub, _ := s.CreateFolder(ctx(), 1, "sub", "", &root.ID)
	n, _ := s.CreateBoard(ctx(), 1, "deep", &sub.ID)

	// Расшарить КОРНЕВУЮ папку пользователю 2 на чтение — доступ каскадит вниз.
	if _, err := s.ShareFolder(ctx(), 1, root.ID, domain.TargetUser, 2, false); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetBoard(ctx(), 2, n.ID)
	if err != nil {
		t.Fatalf("shared read failed: %v", err)
	}
	if got.MyAccess != domain.AccessView {
		t.Fatalf("want view, got %q", got.MyAccess)
	}
	// Без права правки — обновление отклоняется.
	title := "hack"
	if _, err := s.UpdateBoard(ctx(), 2, n.ID, domain.BoardUpdate{Title: &title}); err != domain.ErrMemberReadOnly {
		t.Fatalf("want read-only, got %v", err)
	}
}

func TestSharedListIncludesFolderSharedBoards(t *testing.T) {
	repo, users := newFakeRepo(), newFakeUsers()
	users.users[1] = &domain.User{ID: 1, IsActive: true}
	users.users[2] = &domain.User{ID: 2, FIO: "B", IsActive: true}
	s := newSvc(repo, users)
	fol, _ := s.CreateFolder(ctx(), 1, "f", "", nil)
	s.CreateBoard(ctx(), 1, "n", &fol.ID)
	s.ShareFolder(ctx(), 1, fol.ID, domain.TargetUser, 2, true)

	list, err := s.ListSharedBoards(ctx(), 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 shared board, got %d", len(list))
	}
}

// TestRecipientPlacesSharedBoardInOwnFolder — адресат раскладывает чужую доску
// по своей папке и в личный архив; у владельца ничего не меняется.
func TestRecipientPlacesSharedBoardInOwnFolder(t *testing.T) {
	repo, users := newFakeRepo(), newFakeUsers()
	users.users[1] = &domain.User{ID: 1, FIO: "Owner", IsActive: true}
	users.users[2] = &domain.User{ID: 2, FIO: "Recip", IsActive: true}
	s := newSvc(repo, users)

	n, _ := s.CreateBoard(ctx(), 1, "shared", nil)
	s.ShareBoard(ctx(), 1, n.ID, domain.TargetUser, 2, false)
	myFolder, _ := s.CreateFolder(ctx(), 2, "mine", "", nil)

	// До размещения — в «Поделились со мной».
	if list, _ := s.ListSharedBoards(ctx(), 2, ""); len(list) != 1 {
		t.Fatalf("want 1 shared before placing, got %d", len(list))
	}

	// Разместить в своей папке.
	moved, err := s.MoveBoard(ctx(), 2, n.ID, &myFolder.ID)
	if err != nil {
		t.Fatal(err)
	}
	if moved.FolderID == nil || *moved.FolderID != myFolder.ID || moved.MyAccess != domain.AccessView {
		t.Fatalf("bad moved board: folder=%v access=%q", moved.FolderID, moved.MyAccess)
	}
	// У владельца папка не изменилась.
	if repo.boards[n.ID].FolderID != nil {
		t.Fatal("owner board folder changed")
	}
	// Ушла из «Поделились», появилась в моей папке.
	if list, _ := s.ListSharedBoards(ctx(), 2, ""); len(list) != 0 {
		t.Fatalf("want 0 shared after placing, got %d", len(list))
	}
	inFolder, _ := s.ListBoards(ctx(), 2, ListBoardsParams{FolderID: &myFolder.ID, FolderSet: true})
	if len(inFolder) != 1 || inFolder[0].OwnerName != "Owner" {
		t.Fatalf("shared board not in my folder: %+v", inFolder)
	}

	// Личный архив «только у меня».
	arch := true
	if _, err := s.UpdateBoard(ctx(), 2, n.ID, domain.BoardUpdate{Archived: &arch}); err != nil {
		t.Fatal(err)
	}
	if repo.boards[n.ID].Archived {
		t.Fatal("owner board archived — must be personal only")
	}
	inArchive, _ := s.ListBoards(ctx(), 2, ListBoardsParams{Archived: true})
	if len(inArchive) != 1 {
		t.Fatalf("want 1 in my archive, got %d", len(inArchive))
	}
	if again, _ := s.ListBoards(ctx(), 2, ListBoardsParams{FolderID: &myFolder.ID, FolderSet: true}); len(again) != 0 {
		t.Fatalf("archived board still in folder: %d", len(again))
	}
}

// TestRecipientPlacesSharedFolder — адресат подшивает чужую расшаренную папку под
// свою; она уходит из shared-корней и попадает в мои folders с моим parent_id.
func TestRecipientPlacesSharedFolder(t *testing.T) {
	repo, users := newFakeRepo(), newFakeUsers()
	users.users[1] = &domain.User{ID: 1, FIO: "Owner", IsActive: true}
	users.users[2] = &domain.User{ID: 2, FIO: "Recip", IsActive: true}
	s := newSvc(repo, users)

	shared, _ := s.CreateFolder(ctx(), 1, "theirs", "", nil)
	s.ShareFolder(ctx(), 1, shared.ID, domain.TargetUser, 2, true)
	mine, _ := s.CreateFolder(ctx(), 2, "mine", "", nil)

	tree, _ := s.ListFolders(ctx(), 2)
	if len(tree.Shared) != 1 {
		t.Fatalf("want 1 shared root before, got %d", len(tree.Shared))
	}

	if _, err := s.MoveFolder(ctx(), 2, shared.ID, &mine.ID); err != nil {
		t.Fatal(err)
	}
	if repo.folders[shared.ID].ParentID != nil {
		t.Fatal("owner folder parent changed")
	}
	tree, _ = s.ListFolders(ctx(), 2)
	if len(tree.Shared) != 0 {
		t.Fatalf("shared root still present: %d", len(tree.Shared))
	}
	var placed *domain.Folder
	for _, f := range tree.Folders {
		if f.ID == shared.ID {
			placed = f
		}
	}
	if placed == nil || placed.ParentID == nil || *placed.ParentID != mine.ID || placed.OwnerName != "Owner" {
		t.Fatalf("shared folder not placed under mine: %+v", placed)
	}
}

var _ = json.Marshal

func (f *fakeRepo) CountBoards(_ context.Context, ownerID int64) (int, error) {
	n := 0
	for _, b := range f.boards {
		if b.OwnerID == ownerID {
			n++
		}
	}
	return n, nil
}
