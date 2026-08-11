package service

import (
	"context"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

func textFields() []domain.Field {
	return []domain.Field{{ID: 10, Label: "Имя", Type: domain.FieldText}}
}

// Ссылка на просмотр остаётся read-only, даже если клиент постучится в
// пишущую ручку напрямую.
func TestSharedWrite_RejectedForViewLink(t *testing.T) {
	svc, repo, bus := newTestService(textFields())
	repo.share = &domain.Share{ID: 1, RegistryID: 1, Code: "abc", Access: domain.ShareView}

	_, err := svc.SharedCreateRecord(context.Background(), "abc", map[string]any{"10": "Аня"})
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
	repo.share = &domain.Share{ID: 1, RegistryID: 1, Code: "abc", Access: domain.ShareEdit}

	rec, err := svc.SharedCreateRecord(context.Background(), "abc", map[string]any{"10": "Аня"})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if rec.CreatedBy != nil {
		t.Errorf("у записи по ссылке автора нет, получено %v", *rec.CreatedBy)
	}
	if rec.Data["10"] != "Аня" {
		t.Errorf("значение поля не сохранено: %v", rec.Data)
	}
	// Компанию по коду ссылки берём у реестра — иначе событие не дойдёт до
	// сотрудников, у которых раздел открыт.
	if len(bus.events) != 1 || bus.events[0] != "record:created" {
		t.Errorf("ожидалось record:created, получено %v", bus.events)
	}
}

func TestSharedUpdateRecord_UnknownCodeIsNotFound(t *testing.T) {
	svc, _, _ := newTestService(textFields())

	_, err := svc.SharedUpdateRecord(context.Background(), "нет-такого", 1, map[string]any{})
	if de := domain.AsDomainError(err); de == nil || de.HTTPStatus != 404 {
		t.Fatalf("ожидалась 404 по неизвестному коду, получено %v", err)
	}
}

func TestCreateShare_NormalizesAccess(t *testing.T) {
	svc, _, _ := newTestService(textFields())

	edit, err := svc.CreateShare(context.Background(), 7, 1, 42, domain.ShareEdit)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if edit.Access != domain.ShareEdit {
		t.Errorf("уровень ссылки %q, ожидался edit", edit.Access)
	}

	// Незнакомое значение — не повод выдать лишние права.
	weird, err := svc.CreateShare(context.Background(), 7, 1, 42, "admin")
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if weird.Access != domain.ShareView {
		t.Errorf("уровень ссылки %q, ожидался view", weird.Access)
	}
}

// Замена картинки не должна оставлять прежний файл и его миниатюру в квоте.
func TestUpdateRecord_RemovesReplacedFiles(t *testing.T) {
	fields := []domain.Field{{ID: 13, Label: "Фото", Type: domain.FieldImage}}
	svc, repo, _ := newTestService(fields)
	repo.records = map[int64]*domain.Record{
		5: {ID: 5, RegistryID: 1, Data: map[string]any{
			"13": map[string]any{"path": "registry/old.png", "thumb": "registry/old-thumb.jpg"},
		}},
	}
	files := svc.files.(*fakeFiles)

	if _, err := svc.UpdateRecord(context.Background(), 7, 1, 5, map[string]any{
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
