package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/billing/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/billing/internal/payments"
)

// fakeOwners — сервисы-владельцы файлов: что у них живо и что они дают удалить.
type fakeOwners struct {
	files map[string][]*domain.OwnedFile // service → живые файлы
	fail  map[string]bool                // service → недоступен
	// deleted — что у кого попросили удалить (проверяем адресность).
	deleted map[string][]string
}

func (f *fakeOwners) Services() []string {
	return []string{"messenger", "notes"}
}

func (f *fakeOwners) ListFiles(_ domain.Ctx, service string, _ int64, _ []int64) ([]*domain.OwnedFile, error) {
	if f.fail[service] {
		return nil, errors.New("владелец недоступен")
	}
	return f.files[service], nil
}

func (f *fakeOwners) DeleteFiles(_ domain.Ctx, service string, _ int64, _ []int64, keys []string) ([]string, error) {
	if f.fail[service] {
		return nil, errors.New("владелец недоступен")
	}
	if f.deleted == nil {
		f.deleted = map[string][]string{}
	}
	f.deleted[service] = append(f.deleted[service], keys...)
	kept := []*domain.OwnedFile{}
	for _, file := range f.files[service] {
		if !contains(keys, file.Key) {
			kept = append(kept, file)
		}
	}
	f.files[service] = kept
	return keys, nil
}

// fakeObjects — хранилище: размеры незнакомых файлов и удаление сирот.
type fakeObjects struct {
	sizes   map[string]int64
	removed []string
}

func (o *fakeObjects) Size(_ domain.Ctx, key string) (int64, error) {
	size, ok := o.sizes[key]
	if !ok {
		return 0, errors.New("нет такого объекта")
	}
	return size, nil
}

func (o *fakeObjects) Remove(_ domain.Ctx, keys ...string) {
	o.removed = append(o.removed, keys...)
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func newStorageService(t *testing.T) (*Service, *fakeRepo, *fakeOwners, *fakeObjects) {
	t.Helper()
	repo := newRepo()
	owners := &fakeOwners{files: map[string][]*domain.OwnedFile{}, fail: map[string]bool{}}
	objects := &fakeObjects{sizes: map[string]int64{}}
	svc := New(Deps{
		Catalog: repo, Subs: repo, Orders: repo, Promos: repo, Products: repo,
		AI: repo, Storage: repo, Settings: repo, Audit: repo,
		Identity: &fakeIdentity{owners: map[int64]int64{}},
		Provider: payments.NewManual(),
		Owners:   owners, Objects: objects,
		Log: slog.New(slog.DiscardHandler),
	})
	return svc, repo, owners, objects
}

/* Сверка доучитывает файлы, о которых журнал не знал (всё, что залито до его
   появления), и снимает сирот — записи, за которыми уже не стоит ничья
   сущность. Счётчик по итогу равен сумме журнала. */
func TestSweepStorageAddsUnknownAndDropsOrphans(t *testing.T) {
	svc, repo, owners, objects := newStorageService(t)
	ctx := context.Background()

	// Журнал помнит файл, которого у владельца уже нет.
	if err := repo.AddFiles(ctx, 1, []*domain.StoredFile{
		{Key: "messenger/orphan.png", Service: "messenger", Size: 500},
	}); err != nil {
		t.Fatalf("подготовка журнала: %v", err)
	}
	repo.storage[1] = 500

	// А у владельца лежит файл, которого журнал не знает.
	owners.files["notes"] = []*domain.OwnedFile{{Key: "notes/pic.png", Name: "pic.png"}}
	objects.sizes["notes/pic.png"] = 1200

	out, err := svc.SweepStorage(ctx, 1)
	if err != nil {
		t.Fatalf("SweepStorage: %v", err)
	}
	if out.AddedFiles != 1 || out.DeletedFiles != 1 {
		t.Fatalf("ждали +1 доучтённый и −1 сироту: %+v", out)
	}
	if out.UsedBytes != 1200 {
		t.Fatalf("счётчик должен сойтись с журналом (1200), получили %d", out.UsedBytes)
	}
	if !contains(objects.removed, "messenger/orphan.png") {
		t.Fatalf("сирота должна уйти из хранилища: %v", objects.removed)
	}
}

/* Недоступный владелец не должен превращать свои файлы в сирот: его молчание
   нельзя принять за «файлов нет», иначе сверка снесла бы их все. */
func TestSweepStorageKeepsFilesOfUnavailableOwner(t *testing.T) {
	svc, repo, owners, objects := newStorageService(t)
	ctx := context.Background()

	if err := repo.AddFiles(ctx, 1, []*domain.StoredFile{
		{Key: "messenger/live.png", Service: "messenger", Size: 700},
	}); err != nil {
		t.Fatalf("подготовка журнала: %v", err)
	}
	owners.fail["messenger"] = true

	out, err := svc.SweepStorage(ctx, 1)
	if err != nil {
		t.Fatalf("SweepStorage: %v", err)
	}
	if out.DeletedFiles != 0 || len(objects.removed) != 0 {
		t.Fatalf("файлы недоступного владельца трогать нельзя: %+v / %v", out, objects.removed)
	}
	if out.UsedBytes != 700 {
		t.Fatalf("место обязано остаться учтённым, получили %d", out.UsedBytes)
	}
}

// Удаление адресуется владельцу ключа: чужого сервиса оно не касается.
func TestDeleteStorageFilesRoutesToOwner(t *testing.T) {
	svc, repo, owners, _ := newStorageService(t)
	ctx := context.Background()

	if err := repo.AddFiles(ctx, 1, []*domain.StoredFile{
		{Key: "notes/a.png", Service: "notes", Size: 300},
		{Key: "messenger/b.png", Service: "messenger", Size: 200},
	}); err != nil {
		t.Fatalf("подготовка журнала: %v", err)
	}
	repo.storage[1] = 500

	out, err := svc.DeleteStorageFiles(ctx, 1, []string{"notes/a.png"})
	if err != nil {
		t.Fatalf("DeleteStorageFiles: %v", err)
	}
	if out.DeletedFiles != 1 || out.FreedBytes != 300 {
		t.Fatalf("ждали один файл на 300 байт: %+v", out)
	}
	if len(owners.deleted["messenger"]) != 0 {
		t.Fatalf("мессенджера просить было не о чем: %v", owners.deleted)
	}
	if out.UsedBytes != 200 {
		t.Fatalf("счётчик должен уменьшиться до 200, получили %d", out.UsedBytes)
	}
	// Запись журнала уходит вместе с файлом.
	if files, _ := repo.AllFiles(ctx, 1); len(files) != 1 {
		t.Fatalf("в журнале должна остаться одна запись: %+v", files)
	}
}

// Ключ, которого нет в журнале пользователя, до владельца не доходит: иначе
// чужой файл можно было бы удалить, зная его имя.
func TestDeleteStorageFilesIgnoresForeignKeys(t *testing.T) {
	svc, _, owners, _ := newStorageService(t)

	out, err := svc.DeleteStorageFiles(context.Background(), 1, []string{"notes/чужой.png"})
	if err != nil {
		t.Fatalf("DeleteStorageFiles: %v", err)
	}
	if out.DeletedFiles != 0 || len(owners.deleted) != 0 {
		t.Fatalf("чужой ключ не должен уходить владельцу: %+v / %v", out, owners.deleted)
	}
}
