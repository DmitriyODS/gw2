package service

import (
	"context"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/dto"
)

func TestCreateFolderAndList(t *testing.T) {
	svc, _, _, pub := newTestEnv()
	ctx := context.Background()

	f, err := svc.CreateFolder(ctx, 2, dto.FolderInput{Title: "  Работа  ", Emoji: str("💼"), IncludeGroups: true})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if f.Title != "Работа" {
		t.Fatalf("title not trimmed: %q", f.Title)
	}
	if !f.IncludeGroups || f.IncludePersonal {
		t.Fatalf("flags wrong: %+v", f)
	}
	if len(pub.byName("folders:changed")) == 0 {
		t.Fatalf("expected folders:changed event")
	}

	list, err := svc.ListFolders(ctx, 2)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}
}

func TestCreateFolderTitleRequired(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	if _, err := svc.CreateFolder(context.Background(), 2, dto.FolderInput{Title: "   "}); err == nil {
		t.Fatalf("expected error for empty title")
	}
}

func TestFolderLimit(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()
	for i := 0; i < maxFolders; i++ {
		if _, err := svc.CreateFolder(ctx, 2, dto.FolderInput{Title: "f"}); err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
	}
	if _, err := svc.CreateFolder(ctx, 2, dto.FolderInput{Title: "over"}); err == nil {
		t.Fatalf("expected limit error")
	}
}

// Папки скоупятся по владельцу: чужую не видно, не изменить, не удалить.
func TestFolderScopedToOwner(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()

	f, err := svc.CreateFolder(ctx, 2, dto.FolderInput{Title: "Алисина"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Боб (id 3) не видит папку Алисы.
	list, _ := svc.ListFolders(ctx, 3)
	if len(list) != 0 {
		t.Fatalf("bob should see no folders, got %d", len(list))
	}
	// Боб не может удалить чужую папку — она остаётся у Алисы.
	if err := svc.DeleteFolder(ctx, 3, f.ID); err != nil {
		t.Fatalf("delete by non-owner should be no-op, got %v", err)
	}
	if list, _ := svc.ListFolders(ctx, 2); len(list) != 1 {
		t.Fatalf("alice folder must survive foreign delete")
	}
}

func TestReorderFolders(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()
	a, _ := svc.CreateFolder(ctx, 2, dto.FolderInput{Title: "A"})
	b, _ := svc.CreateFolder(ctx, 2, dto.FolderInput{Title: "B"})

	if err := svc.ReorderFolders(ctx, 2, []int64{b.ID, a.ID}); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	list, _ := svc.ListFolders(ctx, 2)
	if list[0].ID != b.ID || list[1].ID != a.ID {
		t.Fatalf("reorder not applied: %v", []int64{list[0].ID, list[1].ID})
	}
}

func TestUpdateFolderReplacesItems(t *testing.T) {
	svc, _, _, _ := newTestEnv()
	ctx := context.Background()

	// Реальный диалог Алисы (2) с Бобом (3).
	conv, err := svc.OpenConversation(ctx, 2, 3)
	if err != nil {
		t.Fatalf("open conv: %v", err)
	}

	f, _ := svc.CreateFolder(ctx, 2, dto.FolderInput{Title: "Люди", ConversationIDs: &[]int64{conv.ID}})
	if len(f.ConversationIDs) != 1 || f.ConversationIDs[0] != conv.ID {
		t.Fatalf("items not set on create: %+v", f.ConversationIDs)
	}

	// Пустой список — снимаем все привязки.
	upd, err := svc.UpdateFolder(ctx, 2, f.ID, dto.FolderInput{Title: "Люди", ConversationIDs: &[]int64{}})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(upd.ConversationIDs) != 0 {
		t.Fatalf("items not cleared: %+v", upd.ConversationIDs)
	}

	// nil-состав при правке полей не трогает привязки.
	_, _ = svc.UpdateFolder(ctx, 2, f.ID, dto.FolderInput{Title: "Люди", ConversationIDs: &[]int64{conv.ID}})
	upd2, _ := svc.UpdateFolder(ctx, 2, f.ID, dto.FolderInput{Title: "Коллеги"})
	if len(upd2.ConversationIDs) != 1 {
		t.Fatalf("nil items must not touch membership: %+v", upd2.ConversationIDs)
	}
	if upd2.Title != "Коллеги" {
		t.Fatalf("title not updated: %q", upd2.Title)
	}
}
