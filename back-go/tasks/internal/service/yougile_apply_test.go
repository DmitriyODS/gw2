package service

// Порт back/tests/test_yougile_webhook_apply.py: маршрутизация событий,
// антицикл, изменения title/deadline/completed, deleted/restored.

import (
	"context"
	"testing"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

// seedLinkedTask — задача, привязанная к карточке yg-1 (как _make_task).
func (e *ygEnv) seedLinkedTask(over func(t *domain.Task)) *domain.Task {
	task := seedTask(e.store, 1)
	task.Name = "Hello"
	task.AuthorID = 99
	task.LinkYougile = ptr("https://ru.yougile.com/x")
	task.YougileTaskID = ptr("yg-1")
	task.YougileProjectID = ptr("p")
	task.YougileBoardID = ptr("board-1")
	task.YougileColumnID = ptr("c1")
	if over != nil {
		over(task)
	}
	return task
}

func applyEnv() (*ygEnv, *domain.YougileCompany) {
	e := newYGEnv()
	company := e.seedCompany(func(c *domain.YougileCompany) {
		c.YgBoardID = ptr("board-1")
		c.YgFirstColumnID = ptr("col-first")
		c.YgCompletedColumnID = nil
	})
	employee(e.users, 99, 1) // автор задач для системных комментариев
	return e, company
}

// ── маршрутизация ────────────────────────────────────────────────

func TestApplySkippedWhenTaskUnknown(t *testing.T) {
	e, company := applyEnv()
	out, err := e.yg.applyEvent(context.Background(), company, map[string]any{
		"event": "task-updated", "data": map[string]any{"id": "yg-x"},
	})
	if err != nil || out["status"] != "skipped" {
		t.Fatalf("out = %v, err = %v", out, err)
	}
}

func TestApplyNoIDSkipped(t *testing.T) {
	e, company := applyEnv()
	out, err := e.yg.applyEvent(context.Background(), company, map[string]any{
		"event": "task-updated",
	})
	if err != nil || out["status"] != "skipped" || out["reason"] != "no-id" {
		t.Fatalf("out = %v, err = %v", out, err)
	}
}

// ── антицикл ─────────────────────────────────────────────────────

func TestSelfEchoIsSkipped(t *testing.T) {
	e, company := applyEnv()
	task := e.seedLinkedTask(func(task *domain.Task) {
		task.YougileSyncHash = ptr(syncHash("Hello", 0, false))
	})
	out, err := e.yg.applyEvent(context.Background(), company, map[string]any{
		"event": "task-updated",
		"data":  map[string]any{"id": "yg-1", "title": "Hello", "completed": false},
	})
	if err != nil || out["status"] != "skipped" || out["reason"] != "self-echo" {
		t.Fatalf("out = %v, err = %v", out, err)
	}
	if task.Name != "Hello" || len(e.bus.events) != 0 {
		t.Fatal("эхо изменило задачу или ушло событие")
	}
}

// ── изменения ────────────────────────────────────────────────────

func TestTitleChangeApplies(t *testing.T) {
	e, company := applyEnv()
	task := e.seedLinkedTask(func(task *domain.Task) { task.Name = "Old" })
	out, err := e.yg.applyEvent(context.Background(), company, map[string]any{
		"event": "task-updated",
		"data":  map[string]any{"id": "yg-1", "title": "New"},
	})
	if err != nil || out["status"] != "applied" {
		t.Fatalf("out = %v, err = %v", out, err)
	}
	fields := out["fields"].([]any)
	if fields[0] != "name" {
		t.Fatalf("fields = %v", fields)
	}
	if task.Name != "New" {
		t.Fatalf("name = %q", task.Name)
	}
	if task.YougileSyncHash == nil || *task.YougileSyncHash == "" {
		t.Fatal("sync_hash не записан")
	}
}

func TestDeadlineAppliesFromMs(t *testing.T) {
	e, company := applyEnv()
	task := e.seedLinkedTask(func(task *domain.Task) { task.Name = "X" })
	const ms = int64(1717000000000)
	out, err := e.yg.applyEvent(context.Background(), company, map[string]any{
		"event": "task-updated",
		"data": map[string]any{"id": "yg-1",
			"deadline": map[string]any{"deadline": float64(ms)}},
	})
	if err != nil || out["status"] != "applied" {
		t.Fatalf("out = %v, err = %v", out, err)
	}
	if task.Deadline == nil || task.Deadline.UnixMilli() != ms {
		t.Fatalf("deadline = %v", task.Deadline)
	}
}

func TestCompletedTriggersArchive(t *testing.T) {
	e, company := applyEnv()
	task := e.seedLinkedTask(func(task *domain.Task) { task.Name = "X" })
	out, err := e.yg.applyEvent(context.Background(), company, map[string]any{
		"event": "task-completed",
		"data":  map[string]any{"id": "yg-1", "title": "X", "completed": true},
	})
	if err != nil || out["status"] != "applied" {
		t.Fatalf("out = %v, err = %v", out, err)
	}
	if !task.IsArchived || task.ArchivedAt == nil {
		t.Fatalf("задача не заархивирована: %+v", task)
	}
}

// Инвариант: задачу с активным юнитом не архивируем даже по completed из YG.
func TestCompletedWithActiveUnitDoesNotArchive(t *testing.T) {
	e, company := applyEnv()
	task := e.seedLinkedTask(func(task *domain.Task) { task.Name = "X" })
	e.store.units[100] = &domain.Unit{ID: 100, TaskID: task.ID, UserID: 5, CompanyID: 1}

	_, err := e.yg.applyEvent(context.Background(), company, map[string]any{
		"event": "task-completed",
		"data":  map[string]any{"id": "yg-1", "title": "X", "completed": true},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if task.IsArchived {
		t.Fatal("задача с активным юнитом заархивирована")
	}
}

func TestMoveToCompletedColumnArchives(t *testing.T) {
	e, company := applyEnv()
	company.YgCompletedColumnID = ptr("done-col")
	task := e.seedLinkedTask(func(task *domain.Task) {
		task.YougileColumnID = ptr("other")
	})
	out, err := e.yg.applyEvent(context.Background(), company, map[string]any{
		"event": "task-moved",
		"data":  map[string]any{"id": "yg-1", "columnId": "done-col", "completed": false},
	})
	if err != nil || out["status"] != "applied" {
		t.Fatalf("out = %v, err = %v", out, err)
	}
	if strOrEmpty(task.YougileColumnID) != "done-col" {
		t.Fatalf("column = %v", task.YougileColumnID)
	}
	if !task.IsArchived {
		t.Fatal("move в completed-колонку не заархивировал")
	}
}

// ── deleted / restored ───────────────────────────────────────────

func TestTaskDeletedUnlinksAndComments(t *testing.T) {
	e, company := applyEnv()
	task := e.seedLinkedTask(func(task *domain.Task) {
		task.LinkYougile = ptr("https://yg/x")
	})
	out, err := e.yg.applyEvent(context.Background(), company, map[string]any{
		"event": "task-deleted",
		"data":  map[string]any{"id": "yg-1"},
	})
	if err != nil || out["status"] != "unlinked" {
		t.Fatalf("out = %v, err = %v", out, err)
	}
	if task.YougileTaskID != nil || task.LinkYougile != nil {
		t.Fatalf("связь не разорвана: %+v", task)
	}
	if len(e.store.comments) != 1 {
		t.Fatalf("комментариев = %d", len(e.store.comments))
	}
	for _, c := range e.store.comments {
		if c.AuthorID != 99 {
			t.Fatalf("комментарий не от автора задачи: %d", c.AuthorID)
		}
	}
}

func TestTaskRestoredUnarchives(t *testing.T) {
	e, company := applyEnv()
	now := time.Now().UTC()
	task := e.seedLinkedTask(func(task *domain.Task) {
		task.IsArchived = true
		task.ArchivedAt = &now
	})
	out, err := e.yg.applyEvent(context.Background(), company, map[string]any{
		"event": "task-restored",
		"data":  map[string]any{"id": "yg-1"},
	})
	if err != nil || out["status"] != "restored" {
		t.Fatalf("out = %v, err = %v", out, err)
	}
	if task.IsArchived || task.ArchivedAt != nil {
		t.Fatalf("архив не снят: %+v", task)
	}
}

// ── HandleWebhook (ингресс) ──────────────────────────────────────

func TestHandleWebhookBadSecret(t *testing.T) {
	e, company := applyEnv()
	company.YgWebhookSecret = ptr("right")
	_, found, err := e.yg.HandleWebhook(context.Background(), 1, "wrong", []byte(`{}`))
	if err != nil || found {
		t.Fatalf("found = %v, err = %v", found, err)
	}
}

func TestHandleWebhookBatchAndSingle(t *testing.T) {
	e, company := applyEnv()
	company.YgWebhookSecret = ptr("s")
	e.seedLinkedTask(func(task *domain.Task) { task.Name = "Old" })

	// Одиночное событие.
	results, found, err := e.yg.HandleWebhook(context.Background(), 1, "s",
		[]byte(`{"event": "task-updated", "data": {"id": "yg-1", "title": "New"}}`))
	if err != nil || !found || len(results) != 1 || results[0]["status"] != "applied" {
		t.Fatalf("results = %v, found = %v, err = %v", results, found, err)
	}

	// Массив событий: сбойное не валит batch.
	results, found, err = e.yg.HandleWebhook(context.Background(), 1, "s",
		[]byte(`[{"event": "task-updated", "data": {"id": "nope"}}, {"event": "task-renamed", "data": {"id": "yg-1", "title": "Third"}}]`))
	if err != nil || !found || len(results) != 2 {
		t.Fatalf("results = %v, found = %v, err = %v", results, found, err)
	}
	if results[0]["status"] != "skipped" || results[1]["status"] != "applied" {
		t.Fatalf("results = %v", results)
	}
}
