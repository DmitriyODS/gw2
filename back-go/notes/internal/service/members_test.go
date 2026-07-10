package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/notes/internal/domain"
)

// ── Адресный доступ: view читает, edit правит title/doc ──

func TestMemberViewEditAccess(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	ctx := context.Background()

	n, _ := svc.CreateNote(ctx, 1, "Общая")
	if _, err := svc.AddMember(ctx, 1, n.ID, 2, false); err != nil {
		t.Fatal(err)
	}

	// view-адресат читает заметку: my_access=view, группы владельца скрыты.
	got, err := svc.GetNote(ctx, 2, n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.MyAccess != domain.AccessView {
		t.Fatalf("my_access адресата: %q", got.MyAccess)
	}
	if got.OwnerName != "Владелец Тест" {
		t.Fatalf("owner_name адресата: %q", got.OwnerName)
	}
	if len(got.GroupIDs) != 0 {
		t.Fatalf("группы владельца утекли адресату: %v", got.GroupIDs)
	}

	// Без can_edit правка запрещена (403).
	title := "правка"
	if _, err := svc.UpdateNote(ctx, 2, n.ID, domain.NoteUpdate{Title: &title}); !errors.Is(err, domain.ErrMemberReadOnly) {
		t.Fatalf("PATCH view-адресатом: ожидался 403, получено %v", err)
	}

	// Upsert меняет право: повторная выдача с can_edit=true.
	if _, err := svc.AddMember(ctx, 1, n.ID, 2, true); err != nil {
		t.Fatal(err)
	}
	if found, canEdit, _ := repo.GetMember(ctx, n.ID, 2); !found || !canEdit {
		t.Fatalf("upsert не поменял право: found=%v can_edit=%v", found, canEdit)
	}
	if _, err := svc.UpdateNote(ctx, 2, n.ID, domain.NoteUpdate{Title: &title, Doc: docWith("текст адресата")}); err != nil {
		t.Fatal(err)
	}
	if repo.notes[n.ID].Title != title || repo.notes[n.ID].TextContent != "текст адресата" {
		t.Fatalf("правка edit-адресата не применилась: %+v", repo.notes[n.ID])
	}

	// Color/archived/pinned адресатом молча игнорируются (личный стиль владельца).
	blue, on := "blue", true
	if _, err := svc.UpdateNote(ctx, 2, n.ID, domain.NoteUpdate{Color: &blue, Archived: &on, Pinned: &on}); err != nil {
		t.Fatal(err)
	}
	if got := repo.notes[n.ID]; got.Color != "" || got.Archived || got.PinnedAt != nil {
		t.Fatalf("адресат поменял поля плитки владельца: color=%q archived=%v pinned=%v", got.Color, got.Archived, got.PinnedAt)
	}

	// Посторонний по-прежнему получает 404.
	if _, err := svc.GetNote(ctx, 5, n.ID); !errors.Is(err, domain.ErrNoteNotFound) {
		t.Fatalf("GetNote посторонним: ожидалась 404, получено %v", err)
	}
	if _, err := svc.UpdateNote(ctx, 5, n.ID, domain.NoteUpdate{Title: &title}); !errors.Is(err, domain.ErrNoteNotFound) {
		t.Fatalf("UpdateNote посторонним: ожидалась 404, получено %v", err)
	}
}

// ── Управление адресатами — только владелец, валидация ──

func TestMembersOwnerOnly(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	ctx := context.Background()

	n, _ := svc.CreateNote(ctx, 1, "Моя")
	_, _ = svc.AddMember(ctx, 1, n.ID, 2, false)

	// Адресат (и посторонний) не управляет шарингом — единая 404.
	if _, err := svc.ListMembers(ctx, 2, n.ID); !errors.Is(err, domain.ErrNoteNotFound) {
		t.Fatalf("ListMembers адресатом: ожидалась 404, получено %v", err)
	}
	if _, err := svc.AddMember(ctx, 2, n.ID, 3, false); !errors.Is(err, domain.ErrNoteNotFound) {
		t.Fatalf("AddMember адресатом: ожидалась 404, получено %v", err)
	}
	if err := svc.RemoveMember(ctx, 2, n.ID, 2); !errors.Is(err, domain.ErrNoteNotFound) {
		t.Fatalf("RemoveMember адресатом: ожидалась 404, получено %v", err)
	}

	// Себе, несуществующему и деактивированному не шарится.
	if _, err := svc.AddMember(ctx, 1, n.ID, 1, false); !errors.Is(err, domain.ErrSelfShare) {
		t.Fatalf("self-share: ожидалась валидация, получено %v", err)
	}
	if _, err := svc.AddMember(ctx, 1, n.ID, 99, false); !errors.Is(err, domain.ErrMemberNotFound) {
		t.Fatalf("несуществующий адресат: ожидалась 404, получено %v", err)
	}
	if _, err := svc.AddMember(ctx, 1, n.ID, 3, false); !errors.Is(err, domain.ErrMemberNotFound) {
		t.Fatalf("деактивированный адресат: ожидалась 404, получено %v", err)
	}
}

// ── Список «поделились со мной» ──

func TestSharedWithMeList(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	ctx := context.Background()

	n, _ := svc.CreateNote(ctx, 1, "Чужая для меня")
	_, _ = svc.AddMember(ctx, 1, n.ID, 2, true)

	shared, err := svc.ListSharedNotes(ctx, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(shared) != 1 || shared[0].ID != n.ID || shared[0].OwnerID != 1 {
		t.Fatalf("shared=1 не отдал чужую заметку: %+v", shared)
	}
	if shared[0].MyAccess != domain.AccessEdit {
		t.Fatalf("my_access в shared-списке: %q", shared[0].MyAccess)
	}

	// У постороннего список пуст.
	other, _ := svc.ListSharedNotes(ctx, 5, "")
	if len(other) != 0 {
		t.Fatalf("shared-список постороннего не пуст: %d", len(other))
	}
}

// ── Collab-броадкаст ──

func TestCollabBroadcast(t *testing.T) {
	svc, repo, bus, _, _ := newTestService()
	ctx := context.Background()

	n, _ := svc.CreateNote(ctx, 1, "Совместная")
	_, _ = svc.AddMember(ctx, 1, n.ID, 2, false)

	// join адресата: комнаты владельца и всех адресатов, в payload есть fio.
	if err := svc.Collab(ctx, 2, n.ID, "join", nil, nil, nil); err != nil {
		t.Fatal(err)
	}
	event, rooms, payload := bus.last(t)
	if event != "note_collab:join" {
		t.Fatalf("событие: %q", event)
	}
	if !slices.Equal(rooms, []string{"user_1", "user_2"}) {
		t.Fatalf("комнаты join: %v", rooms)
	}
	if payload["fio"] != "Адресат Тест" || payload["user_id"] != int64(2) {
		t.Fatalf("payload join: %v", payload)
	}

	// cursor — без fio (клиент кэширует ФИО по user_id из join).
	if err := svc.Collab(ctx, 2, n.ID, "cursor", &domain.CollabCursor{From: 1, To: 5}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, _, payload := bus.last(t); payload["cursor"] == nil || payload["fio"] != nil {
		t.Fatalf("payload cursor: %v", payload)
	}

	// doc view-адресату запрещён; после выдачи can_edit — разрешён.
	if err := svc.Collab(ctx, 2, n.ID, "doc", nil, docWith("x"), nil); !errors.Is(err, domain.ErrMemberReadOnly) {
		t.Fatalf("collab doc view-адресатом: ожидался 403, получено %v", err)
	}
	repo.members[n.ID][2] = true
	if err := svc.Collab(ctx, 2, n.ID, "doc", nil, docWith("x"), nil); err != nil {
		t.Fatal(err)
	}
	// Броадкаст ничего не сохраняет в БД.
	if repo.notes[n.ID].TextContent != "" {
		t.Fatalf("collab doc сохранился в БД: %q", repo.notes[n.ID].TextContent)
	}

	// Название едет в payload вместе с doc (live-обновление у соавторов)…
	newTitle := "Живое название"
	if err := svc.Collab(ctx, 2, n.ID, "doc", nil, docWith("x"), &newTitle); err != nil {
		t.Fatal(err)
	}
	if _, _, payload := bus.last(t); payload["title"] != "Живое название" {
		t.Fatalf("payload doc без title: %v", payload)
	}
	// …но не с cursor (горячий путь без лишних полей) и не пишется в БД.
	if err := svc.Collab(ctx, 2, n.ID, "cursor", nil, nil, &newTitle); err != nil {
		t.Fatal(err)
	}
	if _, _, payload := bus.last(t); payload["title"] != nil {
		t.Fatalf("payload cursor несёт title: %v", payload)
	}
	if repo.notes[n.ID].Title != "Совместная" {
		t.Fatalf("collab title сохранился в БД: %q", repo.notes[n.ID].Title)
	}

	// Посторонний и неизвестный kind — отказ.
	if err := svc.Collab(ctx, 5, n.ID, "cursor", nil, nil, nil); !errors.Is(err, domain.ErrNoteNotFound) {
		t.Fatalf("collab посторонним: ожидалась 404, получено %v", err)
	}
	if err := svc.Collab(ctx, 1, n.ID, "hack", nil, nil, nil); !errors.Is(err, domain.ErrBadCollabKind) {
		t.Fatalf("collab kind=hack: ожидалась валидация, получено %v", err)
	}
}

// ── Адресация сокет-событий: владелец + адресаты ──

func TestEventRoomsIncludeMembers(t *testing.T) {
	svc, _, bus, _, _ := newTestService()
	ctx := context.Background()

	n, _ := svc.CreateNote(ctx, 1, "С событиями")

	// note_member:added — в комнату адресата, с плиткой и правом.
	if _, err := svc.AddMember(ctx, 1, n.ID, 2, true); err != nil {
		t.Fatal(err)
	}
	event, rooms, payload := bus.last(t)
	if event != "note_member:added" || !slices.Equal(rooms, []string{"user_2"}) {
		t.Fatalf("note_member:added: event=%q rooms=%v", event, rooms)
	}
	if payload["can_edit"] != true || payload["note"] == nil {
		t.Fatalf("payload note_member:added: %v", payload)
	}
	if tile := payload["note"].(map[string]any); tile["owner_name"] != "Владелец Тест" {
		t.Fatalf("плитка без владельца: %v", tile)
	}

	// note:updated — владельцу и адресату.
	title := "правка"
	if _, err := svc.UpdateNote(ctx, 1, n.ID, domain.NoteUpdate{Title: &title}); err != nil {
		t.Fatal(err)
	}
	if event, rooms, _ := bus.last(t); event != "note:updated" || !slices.Equal(rooms, []string{"user_1", "user_2"}) {
		t.Fatalf("note:updated: event=%q rooms=%v", event, rooms)
	}

	// note_member:removed — в комнату снятого адресата.
	if err := svc.RemoveMember(ctx, 1, n.ID, 2); err != nil {
		t.Fatal(err)
	}
	if event, rooms, payload := bus.last(t); event != "note_member:removed" ||
		!slices.Equal(rooms, []string{"user_2"}) || payload["note_id"] != n.ID {
		t.Fatalf("note_member:removed: event=%q rooms=%v payload=%v", event, rooms, payload)
	}

	// note:deleted — владельцу и адресатам (адресат снова добавлен).
	_, _ = svc.AddMember(ctx, 1, n.ID, 2, false)
	if err := svc.DeleteNote(ctx, 1, n.ID); err != nil {
		t.Fatal(err)
	}
	if event, rooms, _ := bus.last(t); event != "note:deleted" || !slices.Equal(rooms, []string{"user_1", "user_2"}) {
		t.Fatalf("note:deleted: event=%q rooms=%v", event, rooms)
	}
}

// ── Закрепление: PATCH pinned и порядок в списке ──

func TestPinNote(t *testing.T) {
	svc, repo, _, _, _ := newTestService()
	ctx := context.Background()

	a, _ := svc.CreateNote(ctx, 1, "Обычная")
	b, _ := svc.CreateNote(ctx, 1, "Закреплённая")

	on := true
	got, err := svc.UpdateNote(ctx, 1, b.ID, domain.NoteUpdate{Pinned: &on})
	if err != nil {
		t.Fatal(err)
	}
	if got.PinnedAt == nil {
		t.Fatal("pinned_at не проставлен")
	}

	list, _ := svc.ListNotes(ctx, 1, 0, "", false)
	if len(list) != 2 || list[0].ID != b.ID {
		t.Fatalf("закреплённая не первая в списке: %v, %v", list[0].ID, list[1].ID)
	}
	_ = a

	off := false
	got, err = svc.UpdateNote(ctx, 1, b.ID, domain.NoteUpdate{Pinned: &off})
	if err != nil {
		t.Fatal(err)
	}
	if got.PinnedAt != nil || repo.notes[b.ID].PinnedAt != nil {
		t.Fatal("pinned_at не сброшен")
	}
}

// ── Экспорт в .txt доступен адресату шаринга (чтение есть — выгрузка тоже) ──

func TestExportSharedWithMe(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	ctx := context.Background()

	n, _ := svc.CreateNote(ctx, 1, "Общая")
	doc := docWith("текст для выгрузки")
	if _, err := svc.UpdateNote(ctx, 1, n.ID, domain.NoteUpdate{Doc: doc}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddMember(ctx, 1, n.ID, 2, false); err != nil {
		t.Fatal(err)
	}

	data, name, err := svc.Export(ctx, 2, n.ID)
	if err != nil {
		t.Fatalf("экспорт view-адресатом: %v", err)
	}
	if name != "Общая" || !strings.Contains(string(data), "текст для выгрузки") {
		t.Fatalf("содержимое экспорта: name=%q data=%q", name, data)
	}

	// Посторонний по-прежнему получает 404.
	if _, _, err := svc.Export(ctx, 5, n.ID); !errors.Is(err, domain.ErrNoteNotFound) {
		t.Fatalf("экспорт посторонним: ожидалась 404, получено %v", err)
	}
}
