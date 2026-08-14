package service

import (
	"context"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

func textFields() []domain.Field {
	return []domain.Field{{ID: 10, Label: "Имя", Type: domain.FieldText}}
}

// guestVisit — гость без аккаунта, пришедший по ссылке.
func guestVisit() Visitor {
	return Visitor{IP: "203.0.113.7", UserAgent: "Mozilla/5.0"}
}

// Ссылка на просмотр остаётся read-only, даже если клиент постучится в
// пишущую ручку напрямую.
func TestSharedWrite_RejectedForViewLink(t *testing.T) {
	svc, repo, bus := newTestService(textFields())
	repo.share = &domain.Share{ID: 1, RegistryID: 1, Code: "abc", Access: domain.AccessView}

	_, err := svc.SharedCreateRecord(context.Background(), "abc", guestVisit(), map[string]any{"10": "Аня"})
	if err == nil {
		t.Fatal("ожидался отказ: ссылка только для просмотра")
	}
	if de := domain.AsDomainError(err); de == nil || de.HTTPStatus != 403 {
		t.Fatalf("ожидалась ошибка 403, получено %v", err)
	}
	if len(bus.events) != 0 {
		t.Errorf("отказ не должен порождать события: %v", bus.events)
	}
}

func TestSharedCreateRecord_EditLinkWritesWithoutAuthor(t *testing.T) {
	svc, repo, bus := newTestService(textFields())
	repo.share = &domain.Share{ID: 1, RegistryID: 1, Code: "abc", Access: domain.AccessEdit}

	rec, err := svc.SharedCreateRecord(context.Background(), "abc", guestVisit(), map[string]any{"10": "Аня"})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if rec.CreatedBy != nil {
		t.Errorf("у записи гостя автора нет, получено %v", *rec.CreatedBy)
	}
	if rec.Data["10"] != "Аня" {
		t.Errorf("значение поля не сохранено: %v", rec.Data)
	}
	// Событие адресуется аудитории самого реестра — иначе оно не дойдёт до тех,
	// у кого раздел открыт.
	if len(bus.events) != 1 || bus.events[0] != "record:created" {
		t.Errorf("ожидалось record:created, получено %v", bus.events)
	}
}

// Вошедший по ссылке остаётся автором своей записи: аккаунт у него есть, и
// терять авторство только потому, что он пришёл по коду, незачем.
func TestSharedCreateRecord_KeepsAuthorWhenSignedIn(t *testing.T) {
	svc, repo, _ := newTestService(textFields())
	repo.share = &domain.Share{ID: 1, RegistryID: 1, Code: "abc", Access: domain.AccessEdit}

	user := int64(555)
	v := guestVisit()
	v.UserID = &user
	rec, err := svc.SharedCreateRecord(context.Background(), "abc", v, map[string]any{"10": "Аня"})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if rec.CreatedBy == nil || *rec.CreatedBy != user {
		t.Errorf("автором должен стать вошедший, получено %v", rec.CreatedBy)
	}
}

// Ссылка «только для своих» не отдаёт ничего, пока гость не представился.
func TestSharedRegistry_RequireAuth(t *testing.T) {
	svc, repo, _ := newTestService(textFields())
	repo.share = &domain.Share{
		ID: 1, RegistryID: 1, Code: "abc",
		Access: domain.AccessView, RequireAuth: true,
	}
	ctx := context.Background()

	_, err := svc.SharedRegistry(ctx, "abc", guestVisit())
	if de := domain.AsDomainError(err); de == nil || de.HTTPStatus != 401 {
		t.Fatalf("гостю ожидался отказ 401, получено %v", err)
	}

	user := int64(555)
	v := guestVisit()
	v.UserID = &user
	if _, err := svc.SharedRegistry(ctx, "abc", v); err != nil {
		t.Fatalf("вошедшего ссылка должна пускать: %v", err)
	}
}

func TestSharedUpdateRecord_UnknownCodeIsNotFound(t *testing.T) {
	svc, _, _ := newTestService(textFields())

	_, err := svc.SharedUpdateRecord(context.Background(), "нет-такого", guestVisit(), 1, map[string]any{})
	if de := domain.AsDomainError(err); de == nil || de.HTTPStatus != 404 {
		t.Fatalf("ожидалась 404 по неизвестному коду, получено %v", err)
	}
}

func TestCreateShare_NormalizesAccess(t *testing.T) {
	svc, _, _ := newTestService(textFields())
	ctx := context.Background()

	// Все три уровня — законные значения для ссылки.
	for _, want := range []string{domain.AccessView, domain.AccessEdit, domain.AccessAdmin} {
		share, err := svc.CreateShare(ctx, ownerID, 1, ShareParams{Access: want})
		if err != nil {
			t.Fatalf("неожиданная ошибка: %v", err)
		}
		if share.Access != want {
			t.Errorf("уровень ссылки %q, ожидался %q", share.Access, want)
		}
	}

	// Незнакомое значение — не повод выдать лишние права.
	weird, err := svc.CreateShare(ctx, ownerID, 1, ShareParams{Access: "owner"})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if weird.Access != domain.AccessView {
		t.Errorf("уровень ссылки %q, ожидался view", weird.Access)
	}
}

// Замена картинки не должна оставлять прежний файл и его миниатюру в квоте.
func TestUpdateRecord_RemovesReplacedFiles(t *testing.T) {
	fields := []domain.Field{{ID: 13, Label: "Обложка", Type: domain.FieldImage}}
	svc, repo, _ := newTestService(fields)
	repo.records = map[int64]*domain.Record{
		5: {ID: 5, RegistryID: 1, Data: map[string]any{
			"13": map[string]any{"path": "registry/old.png", "thumb": "registry/old-thumb.jpg"},
		}},
	}
	files := svc.files.(*fakeFiles)

	if _, err := svc.UpdateRecord(context.Background(), ownerID, 1, 5, map[string]any{
		"13": map[string]any{"path": "registry/new.png", "thumb": "registry/new-thumb.jpg"},
	}); err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if len(files.removed) != 2 {
		t.Fatalf("ожидалось удаление файла и миниатюры, удалено %v", files.removed)
	}
	for _, want := range []string{"registry/old.png", "registry/old-thumb.jpg"} {
		found := false
		for _, got := range files.removed {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Errorf("%s не удалён (удалено %v)", want, files.removed)
		}
	}
}
