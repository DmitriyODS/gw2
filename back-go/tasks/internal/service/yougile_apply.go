package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

// Применение событий вебхука YouGile к GW-задачам (порт task_apply.py).
//
// Поддерживаемые события:
//
//	task-created   — игнор: автомассового создания мы не хотим, импорт
//	                 «новых из YG» — ручной сценарий.
//	task-updated   — поменялось одно из полей. Маппим diff.
//	task-moved     — поменялся columnId; move в yg_completed_column_id —
//	                 архивируем у себя.
//	task-deleted   — разрыв связи + системный комментарий + task:updated.
//	task-restored  — снимаем архив, если делали по deleted.
//	task-renamed / task-completed — нормализуются к общему обработчику.
//
// Антицикл: перед апдейтом считаем тот же sync_hash, что при исходящем
// push'е; совпал с сохранённым у задачи — это наше эхо, пропускаем.

func (y *Yougile) applyEvent(ctx context.Context, company *domain.YougileCompany,
	payload map[string]any) (map[string]any, error) {

	event := strings.TrimSpace(strings.ToLower(mapStr(payload, "event")))
	data, _ := payload["data"].(map[string]any)
	if data == nil {
		data = map[string]any{}
	}
	ygTaskID := mapStr(data, "id")
	if ygTaskID == "" {
		ygTaskID = mapStr(payload, "id")
	}
	if ygTaskID == "" {
		y.log.Warn("yougile.webhook_no_id", "event", event)
		return map[string]any{"status": "skipped", "reason": "no-id"}, nil
	}

	task, err := y.repo.TaskByYougileID(ctx, company.ID, ygTaskID)
	if err != nil {
		return nil, err
	}
	// Карточка, которую мы не знаем (не связана с нашей задачей). Игнорим.
	if task == nil {
		if strings.HasPrefix(event, "task-created") {
			return map[string]any{"status": "skipped", "reason": "unlinked-create"}, nil
		}
		return map[string]any{"status": "skipped", "reason": "not-linked"}, nil
	}

	switch {
	case strings.HasPrefix(event, "task-deleted"):
		return y.applyDeleted(ctx, task)
	case strings.HasPrefix(event, "task-restored"):
		return y.applyRestored(ctx, task)
	case strings.HasPrefix(event, "task-moved") || strings.HasPrefix(event, "task-renamed") ||
		strings.HasPrefix(event, "task-updated") || strings.HasPrefix(event, "task-completed"):
		return y.applyUpdated(ctx, company, task, data)
	}
	return map[string]any{"status": "skipped", "reason": "event:" + event}, nil
}

// applyUpdated — применить изменения title/deadline/completed/columnId.
func (y *Yougile) applyUpdated(ctx context.Context, company *domain.YougileCompany,
	task *domain.Task, data map[string]any) (map[string]any, error) {

	incomingTitle := strings.TrimSpace(mapStr(data, "title"))
	var incomingDeadline *time.Time
	if dl, ok := data["deadline"].(map[string]any); ok {
		incomingDeadline = msToTime(jsonNumber(dl["deadline"]))
	}
	incomingCompleted, _ := data["completed"].(bool)

	hashTitle := incomingTitle
	if hashTitle == "" {
		hashTitle = task.Name
	}
	incomingHash := syncHash(hashTitle, timeToMs(incomingDeadline), incomingCompleted)
	// Антицикл: это наш собственный echo.
	if task.YougileSyncHash != nil && incomingHash == *task.YougileSyncHash {
		return map[string]any{"status": "skipped", "reason": "self-echo"}, nil
	}

	fields := map[string]any{}
	// applied — имена полей в порядке добавления (как list(fields.keys())
	// у dict во Flask — порядок в ответе вебхука детерминирован).
	applied := []any{}
	setField := func(name string, value any) {
		fields[name] = value
		applied = append(applied, name)
	}
	if incomingTitle != "" && incomingTitle != task.Name {
		setField("name", incomingTitle)
	}
	if incomingDeadline != nil &&
		(task.Deadline == nil || !incomingDeadline.Equal(*task.Deadline)) {
		setField("deadline", *incomingDeadline)
	}

	newCol := mapStr(data, "columnId")
	if newCol != "" && newCol != strOrEmpty(task.YougileColumnID) {
		setField("yougile_column_id", newCol)
	}

	newIDShort := mapStr(data, "idTaskProject")
	if newIDShort == "" {
		newIDShort = mapStr(data, "idTaskCommon")
	}
	if newIDShort != "" && newIDShort != strOrEmpty(task.YougileIDShort) {
		setField("yougile_id_short", newIDShort)
	}

	// Архивация по «выполнено» — два сигнала: completed=true либо move в
	// yg_completed_column_id. Любого хватит.
	completedCol := strOrEmpty(company.YgCompletedColumnID)
	shouldArchive := incomingCompleted || (completedCol != "" && newCol == completedCol)
	archivedNow := false
	if shouldArchive && !task.IsArchived {
		// Инвариант GW: нельзя архивировать задачу с активным юнитом — иначе
		// юнит «повиснет» на архивной задаче. Пользователь закроет задачу
		// сам, когда остановит юнит.
		hasActive, err := y.svc.tasks.HasActiveUnit(ctx, task.ID)
		if err != nil {
			return nil, err
		}
		if hasActive {
			y.log.Info("yougile.webhook_archive_skipped_active_unit", "task_id", task.ID)
		} else {
			setField("is_archived", true)
			setField("archived_at", time.Now().UTC())
			archivedNow = true
		}
	} else if !incomingCompleted && task.IsArchived && completedCol == "" {
		// Завершённость снимали в YG (а completed-колонка для авто-архива
		// не задана) — отменяем archive.
		setField("is_archived", false)
		setField("archived_at", nil)
	}

	if len(fields) == 0 && task.YougileSyncHash != nil && incomingHash == *task.YougileSyncHash {
		return map[string]any{"status": "no-changes"}, nil
	}

	setField("yougile_synced_at", time.Now().UTC())
	setField("yougile_sync_hash", incomingHash)
	if err := y.svc.tasks.UpdateTaskFields(ctx, task.ID, fields); err != nil {
		return nil, err
	}

	// Закрытие, пришедшее из YouGile, — тоже опорная точка ленты «Мой Groove».
	if archivedNow {
		task.IsArchived = true
		y.svc.groove.OnTaskClosed(task, 0)
	}

	payload, err := y.broadcastTaskUpdate(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	y.log.Info("yougile.webhook_applied", "task_id", task.ID)
	return map[string]any{"status": "applied", "fields": applied, "payload": payload}, nil
}

// applyDeleted — task-deleted: не удаляем у нас, а разрываем связь и
// оставляем системный комментарий (от лица автора задачи).
func (y *Yougile) applyDeleted(ctx context.Context, task *domain.Task) (map[string]any, error) {
	if task.YougileTaskID == nil || *task.YougileTaskID == "" {
		return map[string]any{"status": "skipped", "reason": "already-unlinked"}, nil
	}

	ygURL := strOrEmpty(task.LinkYougile)
	if err := y.svc.tasks.UpdateTaskFields(ctx, task.ID, unlinkFields()); err != nil {
		return nil, err
	}

	author, err := y.svc.users.GetUser(ctx, task.AuthorID)
	if err == nil && author != nil {
		text := "🔗 Карточка в YouGile удалена, связь разорвана"
		if ygURL != "" {
			text += " (была: " + ygURL + ")"
		}
		y.postSystemComment(ctx, task.ID, author.ID, text)
	}
	if _, err := y.broadcastTaskUpdate(ctx, task.ID); err != nil {
		return nil, err
	}
	y.log.Info("yougile.webhook_deleted_unlinked", "task_id", task.ID)
	return map[string]any{"status": "unlinked"}, nil
}

// applyRestored — task-restored: архивная — снимаем архив. Связь не
// восстанавливаем (это отдельный пользовательский экшен).
func (y *Yougile) applyRestored(ctx context.Context, task *domain.Task) (map[string]any, error) {
	if !task.IsArchived {
		return map[string]any{"status": "no-changes"}, nil
	}
	if err := y.svc.tasks.UpdateTaskFields(ctx, task.ID, map[string]any{
		"is_archived": false, "archived_at": nil,
	}); err != nil {
		return nil, err
	}
	if _, err := y.broadcastTaskUpdate(ctx, task.ID); err != nil {
		return nil, err
	}
	return map[string]any{"status": "restored"}, nil
}

// broadcastTaskUpdate — task:updated в комнату all. Вебхуки идут без
// пользователя — дамп с userID=0 (is_favorite=false/color=null; клиенты
// подмёрджат своё локально).
func (y *Yougile) broadcastTaskUpdate(ctx context.Context, taskID int64) (*dto.Task, error) {
	out, err := y.svc.GetTask(ctx, taskID, 0)
	if err != nil {
		return nil, err
	}
	y.svc.broadcastTask(ctx, "task:updated", *out)
	return out, nil
}
