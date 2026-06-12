package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/yougile"
)

// Импорт/экспорт/отвязка одиночной задачи между Groove Work и YouGile
// (порт task_service.py). Только ручные действия пользователя; обе операции
// пишут системные сообщения: в чат GW-задачи — от лица инициатора
// («🔗 Связано с YouGile: <yg_url>»), в чат YG-карточки — ссылку на GW.

// requireCompanyEnabled — admin включил YG в компании, выбраны
// проект/доска/первая колонка. Иначе создавать/импортить карточки нельзя.
func requireCompanyEnabled(company *domain.YougileCompany) error {
	if !company.UsesYougile {
		return domain.NewError("COMPANY_DISABLED",
			"YouGile-интеграция выключена в настройках компании", 400)
	}
	if company.YgCompanyID == nil || company.YgProjectID == nil ||
		company.YgBoardID == nil || company.YgFirstColumnID == nil {
		return domain.NewError("COMPANY_NOT_CONFIGURED",
			"В настройках компании не выбраны проект/доска", 400)
	}
	return nil
}

// requireUserConnected — клиент пользователя или 412 «подключите YG».
func (y *Yougile) requireUserConnected(ctx context.Context, userID int64) (domain.YougileAPI, error) {
	client, err := y.buildClientForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, domain.NewError("USER_NOT_CONNECTED",
			"Подключите свой YouGile-аккаунт в настройках", 412)
	}
	return client, nil
}

// wrapTaskClientErr — BAD_KEY / YOUGILE_ERROR в операциях с задачами.
func wrapTaskClientErr(err error) error {
	if yougile.IsAuth(err) {
		return domain.NewError("BAD_KEY",
			"Ключ YouGile недействителен, переподключите аккаунт", 400)
	}
	return domain.NewError("YOUGILE_ERROR", "YouGile: "+err.Error(), 400)
}

// userCompany — YouGile-срез компании пользователя или NO_COMPANY.
func (y *Yougile) userCompany(ctx context.Context, user *domain.User) (*domain.YougileCompany, error) {
	if user.CompanyID == nil {
		return nil, domain.NewError("NO_COMPANY", "Пользователь без компании", 400)
	}
	company, err := y.repo.GetYougileCompany(ctx, *user.CompanyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.NewError("NO_COMPANY", "Пользователь без компании", 400)
	}
	return company, nil
}

// ── импорт из YouGile ─────────────────────────────────────────────────────

// ImportTask — создать в Groove Work задачу по ссылке на карточку YouGile.
// После импорта в YG-чат карточки публикуется ссылка на GW-задачу, а в
// GW-чате — системный комментарий со ссылкой на YG. Сокет-событие
// task:created уходит как обычный броадкаст задач (без личного цвета).
func (y *Yougile) ImportTask(ctx context.Context, user *domain.User,
	req dto.YougileImport, origin string) (*dto.Task, error) {

	company, err := y.userCompany(ctx, user)
	if err != nil {
		return nil, err
	}
	if err := requireCompanyEnabled(company); err != nil {
		return nil, err
	}
	client, err := y.requireUserConnected(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	parsed := yougile.ParseTaskURL(req.URL)
	if parsed == nil {
		return nil, domain.NewError("BAD_URL",
			"Не удалось разобрать ссылку на YouGile-карточку", 400)
	}

	// Карточка должна быть из той же компании YG. Длинный URL даёт полный
	// UUID компании, короткий — последние 12 hex. Если в ссылке нет
	// company-части — верим клиенту (он привязан к компании ключом).
	errForeign := domain.NewError("FOREIGN_COMPANY",
		"Эта карточка из другой компании YouGile", 400)
	if parsed.CompanyID != "" && parsed.CompanyID != strOrEmpty(company.YgCompanyID) {
		return nil, errForeign
	}
	ownShort := shortTeamID(strOrEmpty(company.YgCompanyID))
	if parsed.ShortTeamID != "" && ownShort != "" && parsed.ShortTeamID != ownShort {
		return nil, errForeign
	}

	// Короткая ссылка — резолвим idTaskProject в UUID через перебор колонок.
	var yg map[string]any
	taskUUID := parsed.TaskID
	if taskUUID == "" && parsed.ShortTaskID != "" {
		yg, err = client.FindTaskByShortID(*company.YgBoardID, parsed.ShortTaskID, nil)
		if err != nil {
			return nil, wrapTaskClientErr(err)
		}
		id, _ := yg["id"].(string)
		if yg == nil || id == "" {
			return nil, domain.NewError("NOT_FOUND_IN_YG",
				"Карточка "+parsed.ShortTaskID+" не найдена на выбранной доске YouGile", 400)
		}
		taskUUID = strings.ToLower(id)
	}

	// Уже привязана? Возвращаем существующую GW-задачу.
	existing, err := y.repo.TaskByYougileID(ctx, company.ID, taskUUID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		out, err := y.svc.enrichTask(ctx, existing, user.ID)
		if err != nil {
			return nil, err
		}
		return &out, nil
	}

	if yg == nil {
		yg, err = client.GetTask(taskUUID)
		if err != nil {
			return nil, wrapTaskClientErr(err)
		}
	}

	title := strings.TrimSpace(mapStr(yg, "title"))
	if title == "" {
		title = "Без названия"
	}
	idShort := mapStr(yg, "idTaskProject")
	if idShort == "" {
		idShort = mapStr(yg, "idTaskCommon")
	}
	var ygDeadline *time.Time
	if req.PullDeadline {
		if dl, ok := yg["deadline"].(map[string]any); ok {
			ygDeadline = msToTime(jsonNumber(dl["deadline"]))
		}
	}

	ygURL := ygTaskURL(strOrEmpty(company.YgCompanyID), taskUUID, idShort)

	// Сначала создаём GW-задачу (с прежними бизнес-проверками create_task).
	task, err := y.svc.createTaskCore(ctx, user.ID, company.ID, dto.TaskCreate{
		Name:              title,
		LinkYougile:       &ygURL,
		DepartmentID:      req.DepartmentID,
		Deadline:          ygDeadline,
		ResponsibleUserID: req.ResponsibleUserID,
		StageID:           req.StageID,
	})
	if err != nil {
		return nil, err
	}

	// И заполняем структурные YG-поля + sync_hash. project/board id напрямую
	// YG не отдаёт в /tasks/{id} — кешируем заданное в компании (обычно
	// карточка живёт в нашей же доске).
	completed, _ := yg["completed"].(bool)
	fields := map[string]any{
		"yougile_task_id":    taskUUID,
		"yougile_id_short":   normalizeOpt(&idShort),
		"yougile_column_id":  normalizeOpt(ptr(mapStr(yg, "columnId"))),
		"yougile_project_id": company.YgProjectID,
		"yougile_board_id":   company.YgBoardID,
		"yougile_synced_at":  time.Now().UTC(),
		"yougile_sync_hash":  syncHash(title, timeToMs(ygDeadline), completed),
	}
	if err := y.svc.tasks.UpdateTaskFields(ctx, task.ID, fields); err != nil {
		return nil, err
	}

	// Системные сообщения: в YG-карточке — ссылка на GW; в GW-чате — на YG.
	y.postYGLinkBack(client, taskUUID, gwTaskURL(task.ID, origin))
	y.postSystemComment(ctx, task.ID, user.ID, "🔗 Связано с YouGile: "+ygURL)

	y.log.Info("yougile.imported", "task_id", task.ID, "yougile_task_id", taskUUID)

	out, err := y.svc.GetTask(ctx, task.ID, user.ID)
	if err != nil {
		return nil, err
	}
	y.svc.broadcastTask(ctx, "task:created", *out)
	return out, nil
}

// ── экспорт в YouGile ─────────────────────────────────────────────────────

// ExportTask — создать карточку в YouGile из существующей GW-задачи и
// связать их.
func (y *Yougile) ExportTask(ctx context.Context, user *domain.User,
	gwTaskID int64, origin string) (*dto.Task, error) {

	company, err := y.userCompany(ctx, user)
	if err != nil {
		return nil, err
	}
	if err := requireCompanyEnabled(company); err != nil {
		return nil, err
	}
	client, err := y.requireUserConnected(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	task, err := y.svc.tasks.GetTask(ctx, gwTaskID)
	if err != nil {
		return nil, err
	}
	if task == nil || task.CompanyID != company.ID {
		return nil, domain.NewError("NOT_FOUND", "Задача не найдена", 404)
	}
	if task.YougileTaskID != nil && *task.YougileTaskID != "" {
		return nil, domain.NewError("ALREADY_LINKED", "Задача уже связана с YouGile", 400)
	}

	// yg_user_id из аккаунта — «себя в YG» для assigned.
	acc, err := y.repo.GetYougileAccount(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	assigned := []string{}
	if acc != nil && acc.YgUserID != nil && *acc.YgUserID != "" {
		assigned = append(assigned, *acc.YgUserID)
	}

	body := map[string]any{
		"title":    task.Name,
		"columnId": *company.YgFirstColumnID,
		"assigned": assigned,
	}
	if task.Deadline != nil {
		created := task.CreatedAt
		body["deadline"] = map[string]any{
			"deadline":  timeToMs(task.Deadline),
			"startDate": timeToMs(&created),
			"withTime":  false,
		}
	}

	yg, err := client.CreateTask(body)
	if err != nil {
		return nil, wrapTaskClientErr(err)
	}
	ygTaskID := mapStr(yg, "id")
	if ygTaskID == "" {
		return nil, domain.NewError("YOUGILE_ERROR",
			"YouGile не вернул id новой карточки", 400)
	}

	// idTaskProject обычно есть сразу в ответе POST /tasks; если вдруг нет —
	// перечитаем GET'ом (короткая ссылка без него работать не будет).
	idShort := mapStr(yg, "idTaskProject")
	if idShort == "" {
		idShort = mapStr(yg, "idTaskCommon")
	}
	if idShort == "" {
		if fresh, err := client.GetTask(ygTaskID); err == nil {
			idShort = mapStr(fresh, "idTaskProject")
			if idShort == "" {
				idShort = mapStr(fresh, "idTaskCommon")
			}
		}
	}

	ygURL := ygTaskURL(strOrEmpty(company.YgCompanyID), ygTaskID, idShort)
	fields := map[string]any{
		"link_yougile":       ygURL,
		"yougile_task_id":    ygTaskID,
		"yougile_id_short":   normalizeOpt(&idShort),
		"yougile_project_id": company.YgProjectID,
		"yougile_board_id":   company.YgBoardID,
		"yougile_column_id":  *company.YgFirstColumnID,
		"yougile_synced_at":  time.Now().UTC(),
		"yougile_sync_hash":  syncHash(task.Name, timeToMs(task.Deadline), false),
	}
	if err := y.svc.tasks.UpdateTaskFields(ctx, task.ID, fields); err != nil {
		return nil, err
	}

	y.postYGLinkBack(client, ygTaskID, gwTaskURL(task.ID, origin))
	y.postSystemComment(ctx, task.ID, user.ID, "🔗 Карточка создана в YouGile: "+ygURL)
	y.log.Info("yougile.exported", "task_id", task.ID, "yougile_task_id", ygTaskID)

	out, err := y.svc.GetTask(ctx, task.ID, user.ID)
	if err != nil {
		return nil, err
	}
	y.svc.broadcastTask(ctx, "task:updated", *out)
	return out, nil
}

// ── отвязка ───────────────────────────────────────────────────────────────

// UnlinkTask — разорвать связь GW-задачи с YouGile (карточка в YG не
// удаляется). Идемпотентно.
func (y *Yougile) UnlinkTask(ctx context.Context, user *domain.User, gwTaskID int64) (*dto.Task, error) {
	task, err := y.svc.tasks.GetTask(ctx, gwTaskID)
	if err != nil {
		return nil, err
	}
	if task == nil || (user.CompanyID != nil && task.CompanyID != *user.CompanyID) {
		return nil, domain.NewError("NOT_FOUND", "Задача не найдена", 404)
	}
	if task.YougileTaskID == nil || *task.YougileTaskID == "" {
		out, err := y.svc.enrichTask(ctx, task, user.ID)
		if err != nil {
			return nil, err
		}
		return &out, nil // уже отвязана — идемпотентно
	}

	ygURL := strOrEmpty(task.LinkYougile)
	if err := y.svc.tasks.UpdateTaskFields(ctx, task.ID, unlinkFields()); err != nil {
		return nil, err
	}

	text := "🔗 Связь с YouGile разорвана"
	if ygURL != "" {
		text += " (была: " + ygURL + ")"
	}
	y.postSystemComment(ctx, task.ID, user.ID, text)
	y.log.Info("yougile.unlinked", "task_id", task.ID)

	out, err := y.svc.GetTask(ctx, task.ID, user.ID)
	if err != nil {
		return nil, err
	}
	y.svc.broadcastTask(ctx, "task:updated", *out)
	return out, nil
}

// unlinkFields — очистка всей YouGile-привязки задачи.
func unlinkFields() map[string]any {
	return map[string]any{
		"link_yougile":       nil,
		"yougile_task_id":    nil,
		"yougile_id_short":   nil,
		"yougile_project_id": nil,
		"yougile_board_id":   nil,
		"yougile_column_id":  nil,
		"yougile_synced_at":  nil,
		"yougile_sync_hash":  nil,
	}
}

func ptr[T any](v T) *T { return &v }
