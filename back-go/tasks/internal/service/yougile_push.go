package service

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

// Исходящий push изменений GW-задачи в YouGile (порт task_push.py).
//
// Вызывается из Service.UpdateTask / ArchiveTask / RestoreTask после записи.
// Best-effort: если интеграция не включена, юзер не подключён или YG лежит —
// молча логируем; пользовательский запрос на GW-задачу НЕ должен ломаться
// из-за внешнего сервиса.
//
// Асинхронность: сам push выполняется в горутине (HTTP к YouGile может
// занять до таймаута × ретраи) — запрос отвечает сразу, push догоняет в
// фоне, перечитывая задачу по id.
//
// Антицикл: пишем yougile_sync_hash от состояния, которое только что
// отправили — вебхук отбросит своё же эхо.

var pushableFields = map[string]bool{"name": true, "deadline": true}

const pushTimeout = 60 * time.Second

// clientFor — клиент актора; любая ошибка (включая misconfig) → nil + лог.
func (y *Yougile) clientFor(ctx context.Context, actorID int64) domain.YougileAPI {
	client, err := y.buildClientForUser(ctx, actorID)
	if err != nil {
		y.log.Warn("yougile.push_client_unavailable", "user_id", actorID, "error", err)
		return nil
	}
	return client
}

// PushAfterUpdate — отправить в YG только то, что реально поменялось у нас
// и интересует YG. Маппинг GW→YG: name → title, deadline → deadline.deadline
// (ms). Остальное (department/responsible/stage) — внутреннее, в YG не шлём.
func (y *Yougile) PushAfterUpdate(taskID, actorID int64, changed []string) {
	relevant := map[string]bool{}
	for _, f := range changed {
		if pushableFields[f] {
			relevant[f] = true
		}
	}
	if len(relevant) == 0 {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), pushTimeout)
		defer cancel()
		y.runAfterUpdate(ctx, taskID, actorID, relevant)
	}()
}

func (y *Yougile) runAfterUpdate(ctx context.Context, taskID, actorID int64, changed map[string]bool) {
	task, err := y.svc.tasks.GetTask(ctx, taskID)
	if err != nil || task == nil || task.YougileTaskID == nil || *task.YougileTaskID == "" {
		return
	}
	client := y.clientFor(ctx, actorID)
	if client == nil {
		return
	}

	body := map[string]any{}
	if changed["name"] {
		body["title"] = task.Name
	}
	if changed["deadline"] {
		if task.Deadline != nil {
			body["deadline"] = map[string]any{
				"deadline": timeToMs(task.Deadline), "startDate": nil, "withTime": false,
			}
		} else {
			body["deadline"] = nil
		}
	}
	if len(body) == 0 {
		return
	}

	newHash := syncHash(task.Name, timeToMs(task.Deadline), task.IsArchived)

	if _, err := client.UpdateTask(*task.YougileTaskID, body); err != nil {
		y.log.Warn("yougile.push_update_failed", "task_id", taskID, "error", err)
		return
	}

	if err := y.svc.tasks.UpdateTaskFields(ctx, taskID, map[string]any{
		"yougile_synced_at": time.Now().UTC(), "yougile_sync_hash": newHash,
	}); err != nil {
		y.log.Warn("yougile.push_update_save_failed", "task_id", taskID, "error", err)
		return
	}
	y.log.Info("yougile.pushed_update", "task_id", taskID)
}

// PushAfterArchive — архивация/восстановление в GW → completed/columnId в YG.
func (y *Yougile) PushAfterArchive(taskID, actorID int64, archived bool) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), pushTimeout)
		defer cancel()
		y.runAfterArchive(ctx, taskID, actorID, archived)
	}()
}

func (y *Yougile) runAfterArchive(ctx context.Context, taskID, actorID int64, archived bool) {
	task, err := y.svc.tasks.GetTask(ctx, taskID)
	if err != nil || task == nil || task.YougileTaskID == nil || *task.YougileTaskID == "" {
		return
	}
	client := y.clientFor(ctx, actorID)
	if client == nil {
		return
	}
	company, err := y.repo.GetYougileCompany(ctx, task.CompanyID)
	if err != nil {
		y.log.Warn("yougile.push_archive_failed", "task_id", taskID, "error", err)
		return
	}

	body := map[string]any{}
	var targetCol *string

	if archived {
		if company != nil && company.YgCompletedColumnID != nil && *company.YgCompletedColumnID != "" {
			targetCol = company.YgCompletedColumnID
		} else {
			body["completed"] = true
		}
	} else {
		// Восстанавливаем: completed=false. Если задача была перенесена в
		// «выполнено»-колонку — возвращаем её в первую (самое ожидаемое
		// поведение после «Восстановить»).
		body["completed"] = false
		if company != nil && company.YgFirstColumnID != nil && *company.YgFirstColumnID != "" &&
			strOrEmpty(task.YougileColumnID) == strOrEmpty(company.YgCompletedColumnID) {
			targetCol = company.YgFirstColumnID
		}
	}
	if targetCol != nil {
		body["columnId"] = *targetCol
	}
	if len(body) == 0 {
		return
	}

	if _, err := client.UpdateTask(*task.YougileTaskID, body); err != nil {
		y.log.Warn("yougile.push_archive_failed", "task_id", taskID, "error", err)
		return
	}

	fields := map[string]any{
		"yougile_synced_at": time.Now().UTC(),
		"yougile_sync_hash": syncHash(task.Name, timeToMs(task.Deadline), archived),
	}
	if targetCol != nil {
		fields["yougile_column_id"] = *targetCol
	}
	if err := y.svc.tasks.UpdateTaskFields(ctx, taskID, fields); err != nil {
		y.log.Warn("yougile.push_archive_save_failed", "task_id", taskID, "error", err)
		return
	}
	y.log.Info("yougile.pushed_archive", "task_id", taskID, "archived", archived)
}
