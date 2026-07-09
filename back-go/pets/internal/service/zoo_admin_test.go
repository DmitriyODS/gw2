package service

import (
	"context"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// ── Удаление питомца сотрудника администратором компании ────────────

func TestDeleteColleaguePetRequiresAdmin(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.pets.GetOrCreate(ctx, 2, 10)

	for _, level := range []int{0, 1, 2} {
		err := env.svc.DeleteColleaguePet(ctx, level, 2, 10)
		de := domain.AsDomainError(err)
		if de == nil || de.Code != "FORBIDDEN" {
			t.Fatalf("уровень %d: ожидался FORBIDDEN, got %v", level, err)
		}
	}
	if env.pets.byUser[2] == nil {
		t.Fatal("питомец удалён без прав")
	}
}

func TestDeleteColleaguePetOtherCompanyNotFound(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.pets.GetOrCreate(ctx, 2, 20) // питомец чужой компании

	err := env.svc.DeleteColleaguePet(ctx, domain.LevelAdmin, 2, 10)
	de := domain.AsDomainError(err)
	if de == nil || de.Code != "PET_NOT_FOUND" {
		t.Fatalf("ожидался PET_NOT_FOUND, got %v", err)
	}
	if env.pets.byUser[2] == nil {
		t.Fatal("питомец чужой компании удалён")
	}

	// Несуществующий питомец — та же 404.
	err = env.svc.DeleteColleaguePet(ctx, domain.LevelAdmin, 99, 10)
	if de := domain.AsDomainError(err); de == nil || de.Code != "PET_NOT_FOUND" {
		t.Fatalf("ожидался PET_NOT_FOUND для несуществующего, got %v", err)
	}
}

func TestDeleteColleaguePetHappyPathAndRecreate(t *testing.T) {
	env := newEnv()
	ctx := context.Background()
	env.pets.GetOrCreate(ctx, 2, 10)

	if err := env.svc.DeleteColleaguePet(ctx, domain.LevelAdmin, 2, 10); err != nil {
		t.Fatalf("DeleteColleaguePet: %v", err)
	}
	if env.pets.byUser[2] != nil {
		t.Fatal("питомец не удалён")
	}
	found := false
	for _, e := range env.pub.events {
		if e == "pet:deleted" {
			found = true
		}
	}
	if !found {
		t.Fatalf("нет события pet:deleted: %v", env.pub.events)
	}

	// Владелец пересоздаёт питомца штатным GetMyPet (свежее яйцо).
	data, err := env.svc.GetMyPet(ctx, 2, 10)
	if err != nil {
		t.Fatalf("GetMyPet после удаления: %v", err)
	}
	if data.Stage != 0 || data.XP != 0 || data.Kudos != 0 {
		t.Fatalf("питомец пересоздан не с нуля: stage=%d xp=%d kudos=%d", data.Stage, data.XP, data.Kudos)
	}
}
