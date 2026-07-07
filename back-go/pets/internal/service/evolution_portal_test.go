package service

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// fakePortal — синхронный фейк портального клиента (реальный клиент
// fire-and-forget внутри себя, интерфейс зовётся синхронно).
type fakePortal struct {
	calls []portalCall
}

type portalCall struct {
	companyID, authorID int64
	kind, title, body   string
}

func (f *fakePortal) CreateSystemPost(companyID, authorUserID int64, kind, title, body string) {
	f.calls = append(f.calls, portalCall{companyID, authorUserID, kind, title, body})
}

func newPortalEnv(portal domain.PortalClient) (*fakePets, *Service) {
	pets := &fakePets{}
	return pets, New(pets, newFakeShop(), &fakeActivity{}, &fakeUsers{}, fakeCompanies{},
		&fakeWork{}, &fakeDaily{}, &fakePub{}, portal, slog.New(slog.DiscardHandler))
}

// Эволюция (кормление перевело XP через порог стадии) публикует системный
// пост pet_evolved в портал от имени владельца, с названием стадии.
func TestEvolutionCreatesPortalSystemPost(t *testing.T) {
	portal := &fakePortal{}
	pets, svc := newPortalEnv(portal)

	pet, _ := pets.GetOrCreate(context.Background(), 1, 10)
	pet.Kudos = 10
	pet.XP = domain.StageXP[1] - 1 // +FeedXP переведёт через порог «Малыша»
	pet.Name = "Барсик"

	if _, err := svc.FeedPet(context.Background(), 1, 10); err != nil {
		t.Fatalf("FeedPet: %v", err)
	}
	if len(portal.calls) != 1 {
		t.Fatalf("портал должен получить ровно 1 пост, получил %d", len(portal.calls))
	}
	call := portal.calls[0]
	if call.companyID != 10 || call.authorID != 1 {
		t.Errorf("адресация поста: company=%d author=%d", call.companyID, call.authorID)
	}
	if call.kind != "pet_evolved" {
		t.Errorf("system_kind = %q", call.kind)
	}
	if !strings.Contains(call.body, "«Барсик»") || !strings.Contains(call.body, domain.StageTitles[1]) {
		t.Errorf("тело поста без имени/стадии: %q", call.body)
	}
}

// Кормление без пересечения порога стадии постов не публикует.
func TestNoPortalPostWithoutEvolution(t *testing.T) {
	portal := &fakePortal{}
	pets, svc := newPortalEnv(portal)

	pet, _ := pets.GetOrCreate(context.Background(), 1, 10)
	pet.Kudos = 10 // XP 0 → +12 < 40, стадия не растёт

	if _, err := svc.FeedPet(context.Background(), 1, 10); err != nil {
		t.Fatalf("FeedPet: %v", err)
	}
	if len(portal.calls) != 0 {
		t.Fatalf("пост без эволюции: %+v", portal.calls)
	}
}

// nil-клиент портала (не настроен) — эволюция проходит без паники.
func TestEvolutionWithNilPortalClient(t *testing.T) {
	pets, svc := newPortalEnv(nil)

	pet, _ := pets.GetOrCreate(context.Background(), 1, 10)
	pet.Kudos = 10
	pet.XP = domain.StageXP[1] - 1

	data, err := svc.FeedPet(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("FeedPet: %v", err)
	}
	if data.Evolved == nil || !*data.Evolved {
		t.Fatal("эволюция не зафиксирована")
	}
}
