package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// fakeRepo — in-memory реализация порта для тестов бизнес-логики.
type fakeRepo struct {
	notes   map[int64]*domain.Note
	groups  map[int64]*domain.Group
	items   map[int64][]int64 // noteID → groupIDs
	shares  map[string]*domain.Share
	members map[int64]map[int64]bool // noteID → userID → can_edit
	nextID  int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		notes:   map[int64]*domain.Note{},
		groups:  map[int64]*domain.Group{},
		items:   map[int64][]int64{},
		shares:  map[string]*domain.Share{},
		members: map[int64]map[int64]bool{},
	}
}

func (f *fakeRepo) ListNotes(_ domain.Ctx, fl domain.NoteListFilter) ([]*domain.Note, error) {
	out := []*domain.Note{}
	for _, n := range f.notes {
		if n.OwnerID != fl.OwnerID {
			continue
		}
		if fl.GroupID > 0 && !slices.Contains(f.items[n.ID], fl.GroupID) {
			continue
		}
		if n.Archived != fl.Archived {
			continue
		}
		out = append(out, n)
	}
	// Как в SQL: pinned_at DESC NULLS LAST, затем id DESC.
	slices.SortFunc(out, func(a, b *domain.Note) int {
		switch {
		case a.PinnedAt != nil && b.PinnedAt == nil:
			return -1
		case a.PinnedAt == nil && b.PinnedAt != nil:
			return 1
		case a.PinnedAt != nil && b.PinnedAt != nil && !a.PinnedAt.Equal(*b.PinnedAt):
			return b.PinnedAt.Compare(*a.PinnedAt)
		default:
			return int(b.ID - a.ID)
		}
	})
	return out, nil
}
func (f *fakeRepo) GetNote(_ domain.Ctx, id int64) (*domain.Note, error) {
	n := f.notes[id]
	if n != nil {
		n.GroupIDs = f.items[id]
	}
	return n, nil
}
func (f *fakeRepo) CreateNote(_ domain.Ctx, n *domain.Note) error {
	f.nextID++
	n.ID = f.nextID
	f.notes[n.ID] = n
	return nil
}
func (f *fakeRepo) UpdateNote(_ domain.Ctx, n *domain.Note) error {
	if f.notes[n.ID] == nil {
		return errors.New("no note")
	}
	f.notes[n.ID] = n
	return nil
}
func (f *fakeRepo) DeleteNote(_ domain.Ctx, id int64) error {
	delete(f.notes, id)
	delete(f.items, id)
	delete(f.members, id) // каскад по FK
	return nil
}
func (f *fakeRepo) SetNoteGroups(_ domain.Ctx, noteID int64, groupIDs []int64) error {
	f.items[noteID] = groupIDs
	return nil
}
func (f *fakeRepo) ListGroups(_ domain.Ctx, ownerID int64) ([]*domain.Group, error) {
	out := []*domain.Group{}
	for _, g := range f.groups {
		if g.OwnerID == ownerID {
			out = append(out, g)
		}
	}
	return out, nil
}
func (f *fakeRepo) GetGroup(_ domain.Ctx, id int64) (*domain.Group, error) { return f.groups[id], nil }
func (f *fakeRepo) CreateGroup(_ domain.Ctx, g *domain.Group) error {
	f.nextID++
	g.ID = f.nextID
	f.groups[g.ID] = g
	return nil
}
func (f *fakeRepo) UpdateGroup(_ domain.Ctx, id int64, name string) error {
	if g := f.groups[id]; g != nil {
		g.Name = name
	}
	return nil
}
func (f *fakeRepo) DeleteGroup(_ domain.Ctx, id int64) error {
	delete(f.groups, id)
	for noteID, ids := range f.items {
		f.items[noteID] = slices.DeleteFunc(ids, func(g int64) bool { return g == id })
	}
	return nil
}
func (f *fakeRepo) NextGroupPosition(_ domain.Ctx, _ int64) (int, error) { return 1, nil }
func (f *fakeRepo) OwnedGroupIDs(_ domain.Ctx, ownerID int64, ids []int64) ([]int64, error) {
	out := []int64{}
	for _, id := range ids {
		if g := f.groups[id]; g != nil && g.OwnerID == ownerID {
			out = append(out, id)
		}
	}
	return out, nil
}
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
	f.nextID++
	s.ID = f.nextID
	f.shares[s.Code] = s
	return nil
}
func (f *fakeRepo) GetShareByCode(_ domain.Ctx, code string) (*domain.Share, error) {
	return f.shares[code], nil
}
func (f *fakeRepo) DeleteShare(_ domain.Ctx, id, noteID int64) error {
	for code, s := range f.shares {
		if s.ID == id && s.NoteID == noteID {
			delete(f.shares, code)
		}
	}
	return nil
}

func (f *fakeRepo) memberIDsSorted(noteID int64) []int64 {
	return slices.Sorted(maps.Keys(f.members[noteID]))
}
func (f *fakeRepo) ListMembers(_ domain.Ctx, noteID int64) ([]*domain.NoteMember, error) {
	out := []*domain.NoteMember{}
	for _, id := range f.memberIDsSorted(noteID) {
		out = append(out, &domain.NoteMember{UserID: id, CanEdit: f.members[noteID][id]})
	}
	return out, nil
}
func (f *fakeRepo) UpsertMember(_ domain.Ctx, noteID, userID int64, canEdit bool) error {
	if f.members[noteID] == nil {
		f.members[noteID] = map[int64]bool{}
	}
	f.members[noteID][userID] = canEdit
	return nil
}
func (f *fakeRepo) DeleteMember(_ domain.Ctx, noteID, userID int64) error {
	delete(f.members[noteID], userID)
	return nil
}
func (f *fakeRepo) GetMember(_ domain.Ctx, noteID, userID int64) (bool, bool, error) {
	canEdit, ok := f.members[noteID][userID]
	return ok, canEdit, nil
}
func (f *fakeRepo) MemberIDs(_ domain.Ctx, noteID int64) ([]int64, error) {
	return f.memberIDsSorted(noteID), nil
}
func (f *fakeRepo) ListSharedWithMe(_ domain.Ctx, userID int64, search string) ([]*domain.Note, error) {
	out := []*domain.Note{}
	for noteID, m := range f.members {
		canEdit, ok := m[userID]
		n := f.notes[noteID]
		if !ok || n == nil {
			continue
		}
		if search != "" && !strings.Contains(n.Title+" "+n.TextContent, search) {
			continue
		}
		cp := *n
		cp.MyAccess = domain.AccessView
		if canEdit {
			cp.MyAccess = domain.AccessEdit
		}
		cp.GroupIDs = []int64{}
		out = append(out, &cp)
	}
	return out, nil
}

type fakeBus struct {
	events   []string
	rooms    [][]string
	payloads []any
}

func (f *fakeBus) Publish(_ domain.Ctx, event string, rooms []string, payload any) {
	f.events = append(f.events, event)
	f.rooms = append(f.rooms, rooms)
	f.payloads = append(f.payloads, payload)
}

// last — последнее опубликованное событие (имя, комнаты, payload-карта).
func (f *fakeBus) last(t *testing.T) (string, []string, map[string]any) {
	t.Helper()
	if len(f.events) == 0 {
		t.Fatal("событий не публиковалось")
	}
	i := len(f.events) - 1
	payload, _ := f.payloads[i].(map[string]any)
	return f.events[i], f.rooms[i], payload
}

type fakeUsers struct{ users map[int64]*domain.User }

func (f *fakeUsers) GetUser(_ domain.Ctx, id int64) (*domain.User, error) {
	return f.users[id], nil
}

type fakeFiles struct{ removed []string }

func (f *fakeFiles) Save(fileName string, _ []byte) (string, error) { return "notes/" + fileName, nil }
func (f *fakeFiles) Remove(paths []string)                          { f.removed = append(f.removed, paths...) }

type fakeLimiter struct{ deny bool }

func (f *fakeLimiter) Allow(_ domain.Ctx, _ string) bool { return !f.deny }

func newTestService() (*Service, *fakeRepo, *fakeBus, *fakeFiles, *fakeLimiter) {
	repo := newFakeRepo()
	bus := &fakeBus{}
	files := &fakeFiles{}
	limiter := &fakeLimiter{}
	// Пользователи платформы: 1–2 активные, 3 — деактивирован (для адресного шаринга).
	users := &fakeUsers{users: map[int64]*domain.User{
		1: {ID: 1, FIO: "Владелец Тест", IsActive: true},
		2: {ID: 2, FIO: "Адресат Тест", IsActive: true},
		3: {ID: 3, FIO: "Уволенный Тест", IsActive: false},
	}}
	svc := New(Deps{Repo: repo, Users: users, Files: files, Bus: bus, Limiter: limiter, Log: discardLogger()})
	return svc, repo, bus, files, limiter
}

func docWith(text string) json.RawMessage {
	return json.RawMessage(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":` +
		string(mustJSON(text)) + `}]}]}`)
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// ── Скоуп по владельцу ──

func TestOwnerScope(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	ctx := context.Background()

	n, err := svc.CreateNote(ctx, 1, "Моя")
	if err != nil {
		t.Fatal(err)
	}

	// Чужая заметка не читается, не правится и не удаляется — единая 404.
	if _, err := svc.GetNote(ctx, 2, n.ID); !errors.Is(err, domain.ErrNoteNotFound) {
		t.Fatalf("GetNote чужим: ожидалась 404, получено %v", err)
	}
	title := "hack"
	if _, err := svc.UpdateNote(ctx, 2, n.ID, domain.NoteUpdate{Title: &title}); !errors.Is(err, domain.ErrNoteNotFound) {
		t.Fatalf("UpdateNote чужим: ожидалась 404, получено %v", err)
	}
	if err := svc.DeleteNote(ctx, 2, n.ID); !errors.Is(err, domain.ErrNoteNotFound) {
		t.Fatalf("DeleteNote чужим: ожидалась 404, получено %v", err)
	}
	if _, err := svc.ListShares(ctx, 2, n.ID); !errors.Is(err, domain.ErrNoteNotFound) {
		t.Fatalf("ListShares чужим: ожидалась 404, получено %v", err)
	}
}

// ── Пересчёт text_content из doc ──

func TestUpdateRecomputesTextContent(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	ctx := context.Background()

	n, _ := svc.CreateNote(ctx, 1, "")
	if _, err := svc.UpdateNote(ctx, 1, n.ID, domain.NoteUpdate{Doc: docWith("привет мир")}); err != nil {
		t.Fatal(err)
	}
	if got := repo.notes[n.ID].TextContent; got != "привет мир" {
		t.Fatalf("text_content не пересчитан: %q", got)
	}

	// Правка только заголовка не трогает text_content.
	title := "Заголовок"
	if _, err := svc.UpdateNote(ctx, 1, n.ID, domain.NoteUpdate{Title: &title}); err != nil {
		t.Fatal(err)
	}
	if got := repo.notes[n.ID].TextContent; got != "привет мир" {
		t.Fatalf("text_content потерян при правке заголовка: %q", got)
	}
}

// ── Шаринг: edit пишет, view — нет ──

func TestSharedEditWritesViewDoesNot(t *testing.T) {
	svc, _, _, _, limiter := newTestService()
	ctx := context.Background()

	n, _ := svc.CreateNote(ctx, 1, "Общая")
	view, err := svc.CreateShare(ctx, 1, n.ID, domain.AccessView)
	if err != nil {
		t.Fatal(err)
	}
	edit, err := svc.CreateShare(ctx, 1, n.ID, domain.AccessEdit)
	if err != nil {
		t.Fatal(err)
	}

	title := "правка по ссылке"
	if _, err := svc.UpdateSharedNote(ctx, view.Code, &title, nil); !errors.Is(err, domain.ErrReadOnly) {
		t.Fatalf("view-ссылка: ожидался 403, получено %v", err)
	}
	got, err := svc.UpdateSharedNote(ctx, edit.Code, &title, docWith("текст по ссылке"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != title || got.TextContent != "текст по ссылке" {
		t.Fatalf("edit-ссылка не применила правку: %+v", got)
	}

	// Троттлинг: отказ лимитера → 429, правка не применяется.
	limiter.deny = true
	if _, err := svc.UpdateSharedNote(ctx, edit.Code, &title, nil); !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("троттлинг: ожидался 429, получено %v", err)
	}

	// Некорректный режим доступа при создании ссылки.
	if _, err := svc.CreateShare(ctx, 1, n.ID, "admin"); !errors.Is(err, domain.ErrBadAccess) {
		t.Fatalf("access=admin: ожидалась валидация, получено %v", err)
	}
}

// ── Экспорт/импорт txt ──

func TestExportImport(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	ctx := context.Background()

	n, _ := svc.CreateNote(ctx, 1, "Список покупок")
	_, _ = svc.UpdateNote(ctx, 1, n.ID, domain.NoteUpdate{Doc: docWith("хлеб и молоко")})

	data, name, err := svc.Export(ctx, 1, n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if name != "Список покупок" {
		t.Fatalf("имя файла: %q", name)
	}
	if string(data) != "Список покупок\n\nхлеб и молоко" {
		t.Fatalf("содержимое экспорта: %q", data)
	}

	imported, err := svc.Import(ctx, 2, "Импортированная\n\nстрока один\nстрока два")
	if err != nil {
		t.Fatal(err)
	}
	if imported.Title != "Импортированная" {
		t.Fatalf("title импорта: %q", imported.Title)
	}
	if imported.TextContent != "строка один\nстрока два" {
		t.Fatalf("текст импорта: %q", imported.TextContent)
	}
	// Документ — валидные параграфы TipTap.
	if !strings.Contains(string(imported.Doc), `"paragraph"`) {
		t.Fatalf("doc импорта без параграфов: %s", imported.Doc)
	}
}

// ── Группы ──

func TestDeleteGroupKeepsNotes(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	ctx := context.Background()

	g, _ := svc.CreateGroup(ctx, 1, "Работа")
	n, _ := svc.CreateNote(ctx, 1, "В группе")
	if _, err := svc.SetGroups(ctx, 1, n.ID, []int64{g.ID}); err != nil {
		t.Fatal(err)
	}

	if err := svc.DeleteGroup(ctx, 1, g.ID); err != nil {
		t.Fatal(err)
	}
	if repo.notes[n.ID] == nil {
		t.Fatal("удаление группы удалило заметку")
	}
	got, _ := svc.GetNote(ctx, 1, n.ID)
	if len(got.GroupIDs) != 0 {
		t.Fatalf("связи с удалённой группой остались: %v", got.GroupIDs)
	}

	// Чужая группа не удаляется.
	g2, _ := svc.CreateGroup(ctx, 1, "Личное")
	if err := svc.DeleteGroup(ctx, 2, g2.ID); !errors.Is(err, domain.ErrGroupNotFound) {
		t.Fatalf("DeleteGroup чужим: ожидалась 404, получено %v", err)
	}
}

func TestSetGroupsDropsForeign(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	ctx := context.Background()

	mine, _ := svc.CreateGroup(ctx, 1, "Моя")
	foreign, _ := svc.CreateGroup(ctx, 2, "Чужая")
	n, _ := svc.CreateNote(ctx, 1, "")

	got, err := svc.SetGroups(ctx, 1, n.ID, []int64{mine.ID, foreign.ID, 999})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.GroupIDs) != 1 || got.GroupIDs[0] != mine.ID {
		t.Fatalf("чужие группы не отброшены: %v", got.GroupIDs)
	}
}

// ── Удаление заметки чистит файлы ──

func TestDeleteNoteRemovesFiles(t *testing.T) {
	svc, _, _, files, _ := newTestService()
	ctx := context.Background()

	n, _ := svc.CreateNote(ctx, 1, "")
	doc := json.RawMessage(`{"type":"doc","content":[
		{"type":"image","attrs":{"src":"/uploads/notes/abc.png"}},
		{"type":"paragraph","content":[{"type":"text","text":"с картинкой"}]}]}`)
	if _, err := svc.UpdateNote(ctx, 1, n.ID, domain.NoteUpdate{Doc: doc}); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteNote(ctx, 1, n.ID); err != nil {
		t.Fatal(err)
	}
	if len(files.removed) != 1 || files.removed[0] != "notes/abc.png" {
		t.Fatalf("файлы заметки не почищены: %v", files.removed)
	}
}

// ── Цвет плитки ──

func TestNoteColor(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	ctx := context.Background()

	n, _ := svc.CreateNote(ctx, 1, "Цветная")
	blue := "blue"
	if _, err := svc.UpdateNote(ctx, 1, n.ID, domain.NoteUpdate{Color: &blue}); err != nil {
		t.Fatal(err)
	}
	if repo.notes[n.ID].Color != "blue" {
		t.Fatalf("цвет не сохранён: %q", repo.notes[n.ID].Color)
	}

	// Сброс цвета пустой строкой.
	none := ""
	if _, err := svc.UpdateNote(ctx, 1, n.ID, domain.NoteUpdate{Color: &none}); err != nil {
		t.Fatal(err)
	}
	if repo.notes[n.ID].Color != "" {
		t.Fatalf("цвет не сброшен: %q", repo.notes[n.ID].Color)
	}

	// Неизвестный цвет — валидация.
	bad := "magenta"
	if _, err := svc.UpdateNote(ctx, 1, n.ID, domain.NoteUpdate{Color: &bad}); !errors.Is(err, domain.ErrBadColor) {
		t.Fatalf("неизвестный цвет: ожидалась валидация, получено %v", err)
	}
}

// Архив: архивная заметка уходит из основного списка в архивный и возвращается.
func TestArchiveNote(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	ctx := context.Background()
	n, _ := svc.CreateNote(ctx, 1, "В архив")

	on := true
	if _, err := svc.UpdateNote(ctx, 1, n.ID, domain.NoteUpdate{Archived: &on}); err != nil {
		t.Fatalf("archive: %v", err)
	}
	active, _ := svc.ListNotes(ctx, 1, 0, "", false)
	if len(active) != 0 {
		t.Fatalf("архивная заметка осталась в основном списке: %d", len(active))
	}
	archived, _ := svc.ListNotes(ctx, 1, 0, "", true)
	if len(archived) != 1 || !archived[0].Archived {
		t.Fatalf("заметка не попала в архивный список")
	}

	off := false
	if _, err := svc.UpdateNote(ctx, 1, n.ID, domain.NoteUpdate{Archived: &off}); err != nil {
		t.Fatalf("unarchive: %v", err)
	}
	active, _ = svc.ListNotes(ctx, 1, 0, "", false)
	if len(active) != 1 {
		t.Fatalf("заметка не вернулась из архива")
	}
}
