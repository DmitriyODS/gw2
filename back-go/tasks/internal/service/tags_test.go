package service

import (
	"context"
	"testing"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

func TestTagsCRUDAndAssign(t *testing.T) {
	svc, store, _, _, bus, _ := newTestService()
	ctx := context.Background()

	tag, err := svc.CreateTag(ctx, 1, "Срочно", "red")
	if err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	if _, err := svc.CreateTag(ctx, 1, "срочно", "blue"); err == nil {
		t.Fatal("дубль имени (без учёта регистра) должен отклоняться")
	}
	// Тег другой компании невидим и неизменяем.
	if _, err := svc.UpdateTag(ctx, 2, tag.ID, nil, nil); err == nil {
		t.Fatal("тег чужой компании должен отвечать 404")
	}

	task := seedTask(store, 1)
	out, err := svc.SetTaskTags(ctx, task.ID, 1, cid(1), []int64{tag.ID, tag.ID})
	if err != nil {
		t.Fatalf("SetTaskTags: %v", err)
	}
	if len(out.Tags) != 1 || out.Tags[0].Name != "Срочно" {
		t.Fatalf("теги задачи: %+v", out.Tags)
	}
	// Броадкаст task:updated несёт теги (общие для компании).
	last := bus.events[len(bus.events)-1]
	if last.Event != "task:updated" {
		t.Fatalf("событие: %s", last.Event)
	}
	if b, ok := last.Payload.(dto.TaskBroadcast); !ok || len(b.Tags) != 1 {
		t.Fatalf("payload без тегов: %+v", last.Payload)
	}

	// Чужой тег назначить нельзя.
	foreign := &domain.Tag{Name: "Чужой", Color: "blue", CompanyID: 2}
	_ = store.CreateTag(ctx, foreign)
	if _, err := svc.SetTaskTags(ctx, task.ID, 1, cid(1), []int64{foreign.ID}); err == nil {
		t.Fatal("тег чужой компании не должен назначаться")
	}

	// Пустой список снимает теги.
	out, err = svc.SetTaskTags(ctx, task.ID, 1, cid(1), []int64{})
	if err != nil {
		t.Fatalf("SetTaskTags (снятие): %v", err)
	}
	if len(out.Tags) != 0 {
		t.Fatalf("теги не сняты: %+v", out.Tags)
	}

	if err := svc.DeleteTag(ctx, 1, tag.ID); err != nil {
		t.Fatalf("DeleteTag: %v", err)
	}
	tags, _ := svc.ListTags(ctx, 1)
	if len(tags) != 0 {
		t.Fatalf("справочник не пуст: %+v", tags)
	}
}
