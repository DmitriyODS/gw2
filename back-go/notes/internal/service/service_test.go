package service

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// ── Fakes ────────────────────────────────────────────────────────────

type fakeRepo struct {
	notes       map[int64]*domain.Note
	folders     map[int64]*domain.Folder
	tags        map[int64]*domain.Tag
	noteTags    map[int64][]int64
	shares      map[string]*domain.Share
	noteUsers   map[int64]map[int64]bool // noteID → userID → canEdit
	noteCos     map[int64]map[int64]bool // noteID → companyID → canEdit
	folderUsers map[int64]map[int64]bool
	folderCos   map[int64]map[int64]bool
	next        int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		notes: map[int64]*domain.Note{}, folders: map[int64]*domain.Folder{}, tags: map[int64]*domain.Tag{},
		noteTags: map[int64][]int64{}, shares: map[string]*domain.Share{},
		noteUsers: map[int64]map[int64]bool{}, noteCos: map[int64]map[int64]bool{},
		folderUsers: map[int64]map[int64]bool{}, folderCos: map[int64]map[int64]bool{},
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

// ── Заметки ──
func (f *fakeRepo) ListNotes(_ domain.Ctx, fl domain.NoteListFilter) ([]*domain.Note, error) {
	out := []*domain.Note{}
	for _, n := range f.notes {
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
		if len(fl.TagIDs) > 0 {
			hit := false
			for _, t := range f.noteTags[n.ID] {
				if slices.Contains(fl.TagIDs, t) {
					hit = true
					break
				}
			}
			if !hit {
				continue
			}
		}
		out = append(out, n)
	}
	slices.SortFunc(out, func(a, b *domain.Note) int { return int(b.ID - a.ID) })
	return out, nil
}
func (f *fakeRepo) GetNote(_ domain.Ctx, id int64) (*domain.Note, error) {
	n := f.notes[id]
	if n != nil {
		n.TagIDs = f.noteTags[id]
	}
	return n, nil
}
func (f *fakeRepo) CreateNote(_ domain.Ctx, n *domain.Note) error {
	n.ID = f.id()
	f.notes[n.ID] = n
	return nil
}
func (f *fakeRepo) UpdateNote(_ domain.Ctx, n *domain.Note) error { f.notes[n.ID] = n; return nil }
func (f *fakeRepo) DeleteNote(_ domain.Ctx, id int64) error       { delete(f.notes, id); return nil }
func (f *fakeRepo) MoveNote(_ domain.Ctx, id int64, folderID *int64) error {
	if n := f.notes[id]; n != nil {
		n.FolderID = folderID
	}
	return nil
}
func (f *fakeRepo) SetNoteTags(_ domain.Ctx, noteID int64, tagIDs []int64) error {
	f.noteTags[noteID] = tagIDs
	return nil
}
func (f *fakeRepo) SharedByMeNoteIDs(_ domain.Ctx, ids []int64) (map[int64]bool, error) {
	res := map[int64]bool{}
	for _, id := range ids {
		if len(f.noteUsers[id]) > 0 || len(f.noteCos[id]) > 0 {
			res[id] = true
		}
	}
	return res, nil
}
func (f *fakeRepo) ListSharedWithMe(_ domain.Ctx, userID int64, companyIDs []int64, _ string) ([]*domain.Note, error) {
	out := []*domain.Note{}
	for _, n := range f.notes {
		if n.OwnerID == userID {
			continue
		}
		found, _, _ := f.NoteAccess(nil, userID, companyIDs, n.ID, n.FolderID)
		if found {
			out = append(out, n)
		}
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
	for _, n := range f.notes {
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
func (f *fakeRepo) ListTags(_ domain.Ctx, ownerID int64) ([]*domain.Tag, error) {
	out := []*domain.Tag{}
	for _, t := range f.tags {
		if t.OwnerID == ownerID {
			out = append(out, t)
		}
	}
	return out, nil
}
func (f *fakeRepo) GetTag(_ domain.Ctx, id int64) (*domain.Tag, error) { return f.tags[id], nil }
func (f *fakeRepo) CreateTag(_ domain.Ctx, t *domain.Tag) error {
	t.ID = f.id()
	f.tags[t.ID] = t
	return nil
}
func (f *fakeRepo) UpdateTag(_ domain.Ctx, id int64, name, color string) error {
	if t := f.tags[id]; t != nil {
		t.Name, t.Color = name, color
	}
	return nil
}
func (f *fakeRepo) DeleteTag(_ domain.Ctx, id int64) error             { delete(f.tags, id); return nil }
func (f *fakeRepo) NextTagPosition(_ domain.Ctx, _ int64) (int, error) { return 0, nil }
func (f *fakeRepo) OwnedTagIDs(_ domain.Ctx, ownerID int64, ids []int64) ([]int64, error) {
	out := []int64{}
	for _, id := range ids {
		if t := f.tags[id]; t != nil && t.OwnerID == ownerID {
			out = append(out, id)
		}
	}
	return out, nil
}

// ── Публичные ссылки ──
func (f *fakeRepo) ListShares(_ domain.Ctx, noteID int64) ([]*domain.Share, error) {
	out := []*domain.Share{}
	for _, s := range f.shares {
		if s.NoteID == noteID {
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
func (f *fakeRepo) DeleteShare(_ domain.Ctx, id, noteID int64) error {
	for k, s := range f.shares {
		if s.ID == id && s.NoteID == noteID {
			delete(f.shares, k)
		}
	}
	return nil
}

// ── Шаринг заметок ──
func (f *fakeRepo) ListNoteMembers(_ domain.Ctx, _ int64) ([]*domain.Member, error) { return nil, nil }
func (f *fakeRepo) UpsertNoteUserShare(_ domain.Ctx, noteID, userID int64, canEdit bool) error {
	if f.noteUsers[noteID] == nil {
		f.noteUsers[noteID] = map[int64]bool{}
	}
	f.noteUsers[noteID][userID] = canEdit
	return nil
}
func (f *fakeRepo) DeleteNoteUserShare(_ domain.Ctx, noteID, userID int64) error {
	delete(f.noteUsers[noteID], userID)
	return nil
}
func (f *fakeRepo) UpsertNoteCompanyShare(_ domain.Ctx, noteID, companyID int64, _ string, canEdit bool, _ int64) error {
	if f.noteCos[noteID] == nil {
		f.noteCos[noteID] = map[int64]bool{}
	}
	f.noteCos[noteID][companyID] = canEdit
	return nil
}
func (f *fakeRepo) DeleteNoteCompanyShare(_ domain.Ctx, noteID, companyID int64) error {
	delete(f.noteCos[noteID], companyID)
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
func (f *fakeRepo) NoteAudienceUserIDs(_ domain.Ctx, _ int64) ([]int64, error)   { return nil, nil }
func (f *fakeRepo) FolderAudienceUserIDs(_ domain.Ctx, _ int64) ([]int64, error) { return nil, nil }

func (f *fakeRepo) NoteAccess(_ domain.Ctx, userID int64, companyIDs []int64, noteID int64, folderID *int64) (bool, bool, error) {
	found, canEdit := false, false
	mark := func(ok, ce bool) {
		if ok {
			found = true
			canEdit = canEdit || ce
		}
	}
	if ce, ok := f.noteUsers[noteID][userID]; ok {
		mark(true, ce)
	}
	for _, cid := range companyIDs {
		if ce, ok := f.noteCos[noteID][cid]; ok {
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
	return f.NoteAccess(nil, userID, companyIDs, 0, &fid)
}

// Эмбеддинги (ИИ-поиск) — в юнит-тестах не задействованы.
func (f *fakeRepo) UpsertNoteEmbedding(_ domain.Ctx, _, _ int64, _ []float32, _ string) error {
	return nil
}
func (f *fakeRepo) SearchNoteEmbeddings(_ domain.Ctx, _ int64, _ []float32, _ string, _ bool, _ int) ([]int64, error) {
	return nil, nil
}
func (f *fakeRepo) ListNotesByIDs(_ domain.Ctx, _ int64, _ []int64, _ bool) ([]*domain.Note, error) {
	return nil, nil
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

func (nopFiles) Save(_ string, _ []byte) (string, error) { return "notes/x", nil }
func (nopFiles) Remove(_ []string)                       {}

type allowLimiter struct{}

func (allowLimiter) Allow(_ domain.Ctx, _ string) bool { return true }

func newSvc(repo *fakeRepo, users *fakeUsers) *Service {
	return New(Deps{Repo: repo, Users: users, Files: nopFiles{}, Bus: nopBus{}, Limiter: allowLimiter{}, Log: discardLogger()})
}

func ctx() context.Context { return context.Background() }

// ── Тесты ────────────────────────────────────────────────────────────

func TestCreateTagAndAssign(t *testing.T) {
	repo, users := newFakeRepo(), newFakeUsers()
	users.users[1] = &domain.User{ID: 1, IsActive: true}
	s := newSvc(repo, users)

	n, err := s.CreateNote(ctx(), 1, "hi", nil)
	if err != nil {
		t.Fatal(err)
	}
	tag, err := s.CreateTag(ctx(), 1, "work", "blue")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetTags(ctx(), 1, n.ID, []int64{tag.ID}); err != nil {
		t.Fatal(err)
	}
	if got := repo.noteTags[n.ID]; len(got) != 1 || got[0] != tag.ID {
		t.Fatalf("tags not set: %v", got)
	}
}

func TestBadColorRejected(t *testing.T) {
	repo, users := newFakeRepo(), newFakeUsers()
	if _, err := newSvc(repo, users).CreateTag(ctx(), 1, "x", "chartreuse"); err != domain.ErrBadColor {
		t.Fatalf("want ErrBadColor, got %v", err)
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
	n, _ := s.CreateNote(ctx(), 1, "n", &mid.ID)
	if err := s.DeleteFolder(ctx(), 1, mid.ID); err != nil {
		t.Fatal(err)
	}
	if repo.notes[n.ID].FolderID == nil || *repo.notes[n.ID].FolderID != parent.ID {
		t.Fatalf("note not reparented: %v", repo.notes[n.ID].FolderID)
	}
}

func TestShareCompanyRequiresMembership(t *testing.T) {
	repo, users := newFakeRepo(), newFakeUsers()
	users.users[1] = &domain.User{ID: 1, IsActive: true}
	users.members[1] = []int64{10}
	users.compNm[10] = "Acme"
	s := newSvc(repo, users)
	n, _ := s.CreateNote(ctx(), 1, "n", nil)

	// Не член компании 99 → отказ.
	if _, err := s.ShareNote(ctx(), 1, n.ID, domain.TargetCompany, 99, false); err != domain.ErrNotCompanyMember {
		t.Fatalf("want ErrNotCompanyMember, got %v", err)
	}
	// Член компании 10 → успех.
	if _, err := s.ShareNote(ctx(), 1, n.ID, domain.TargetCompany, 10, true); err != nil {
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
	n, _ := s.CreateNote(ctx(), 1, "deep", &sub.ID)

	// Расшарить КОРНЕВУЮ папку пользователю 2 на чтение — доступ каскадит вниз.
	if _, err := s.ShareFolder(ctx(), 1, root.ID, domain.TargetUser, 2, false); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetNote(ctx(), 2, n.ID)
	if err != nil {
		t.Fatalf("shared read failed: %v", err)
	}
	if got.MyAccess != domain.AccessView {
		t.Fatalf("want view, got %q", got.MyAccess)
	}
	// Без права правки — обновление отклоняется.
	title := "hack"
	if _, err := s.UpdateNote(ctx(), 2, n.ID, domain.NoteUpdate{Title: &title}); err != domain.ErrMemberReadOnly {
		t.Fatalf("want read-only, got %v", err)
	}
}

func TestSharedListIncludesFolderSharedNotes(t *testing.T) {
	repo, users := newFakeRepo(), newFakeUsers()
	users.users[1] = &domain.User{ID: 1, IsActive: true}
	users.users[2] = &domain.User{ID: 2, FIO: "B", IsActive: true}
	s := newSvc(repo, users)
	fol, _ := s.CreateFolder(ctx(), 1, "f", "", nil)
	s.CreateNote(ctx(), 1, "n", &fol.ID)
	s.ShareFolder(ctx(), 1, fol.ID, domain.TargetUser, 2, true)

	list, err := s.ListSharedNotes(ctx(), 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 shared note, got %d", len(list))
	}
}

var _ = json.Marshal
