package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/yougile"
)

// Настройки YouGile-интеграции на уровне компании (порт company_service.py).
// Права (DIRECTOR+) проверяет транспорт — сервис принимает уже
// валидированного пользователя.

func settingsDTO(c *domain.YougileCompany) *dto.YougileSettings {
	return &dto.YougileSettings{
		Enabled:             c.UsesYougile,
		WebhookRegistered:   c.YgWebhookID != nil && *c.YgWebhookID != "",
		YgBoardID:           c.YgBoardID,
		YgBoardTitle:        c.YgBoardTitle,
		YgCompanyID:         c.YgCompanyID,
		YgCompanyName:       c.YgCompanyName,
		YgCompletedColumnID: c.YgCompletedColumnID,
		YgFirstColumnID:     c.YgFirstColumnID,
		YgProjectID:         c.YgProjectID,
		YgProjectTitle:      c.YgProjectTitle,
	}
}

// CompanySettings — GET /company-settings. companyID nil (root admin без
// выбранной компании) → пустые настройки, чтобы фронт мог отрендерить визард.
func (y *Yougile) CompanySettings(ctx context.Context, companyID *int64) (*dto.YougileSettings, error) {
	if companyID == nil {
		return &dto.YougileSettings{}, nil
	}
	company, err := y.repo.GetYougileCompany(ctx, *companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return &dto.YougileSettings{}, nil
	}
	return settingsDTO(company), nil
}

// ── каталоги (для админ-визарда) ──────────────────────────────────────────

// requireConnectedClient — клиент актора или ошибка NOT_CONNECTED/BAD_KEY.
func (y *Yougile) requireConnectedClient(ctx context.Context, actorID int64) (domain.YougileAPI, error) {
	client, err := y.buildClientForUser(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, domain.NewError("NOT_CONNECTED",
			"Сначала подключите свой YouGile-аккаунт", 400)
	}
	return client, nil
}

// wrapCatalogErr — ошибки клиента YG в каталогах: BAD_KEY / YOUGILE_ERROR.
func wrapCatalogErr(err error) error {
	if yougile.IsAuth(err) {
		return domain.NewError("BAD_KEY",
			"Ключ YouGile недействителен, переподключите аккаунт", 400)
	}
	return domain.NewError("YOUGILE_ERROR", "YouGile: "+err.Error(), 400)
}

func (y *Yougile) Projects(ctx context.Context, actorID int64) ([]dto.YougileProject, error) {
	client, err := y.requireConnectedClient(ctx, actorID)
	if err != nil {
		return nil, err
	}
	items, err := client.ListProjects(1000)
	if err != nil {
		return nil, wrapCatalogErr(err)
	}
	out := make([]dto.YougileProject, 0, len(items))
	for _, i := range items {
		if id, _ := i["id"].(string); id != "" {
			title, _ := i["title"].(string)
			out = append(out, dto.YougileProject{ID: id, Title: title})
		}
	}
	return out, nil
}

func (y *Yougile) Boards(ctx context.Context, actorID int64, projectID string) ([]dto.YougileBoard, error) {
	client, err := y.requireConnectedClient(ctx, actorID)
	if err != nil {
		return nil, err
	}
	items, err := client.ListBoards(projectID, 1000)
	if err != nil {
		return nil, wrapCatalogErr(err)
	}
	out := make([]dto.YougileBoard, 0, len(items))
	for _, i := range items {
		if id, _ := i["id"].(string); id != "" {
			title, _ := i["title"].(string)
			b := dto.YougileBoard{ID: id, Title: title}
			if pid, _ := i["projectId"].(string); pid != "" {
				b.ProjectID = &pid
			}
			out = append(out, b)
		}
	}
	return out, nil
}

func (y *Yougile) Columns(ctx context.Context, actorID int64, boardID string) ([]dto.YougileColumn, error) {
	client, err := y.requireConnectedClient(ctx, actorID)
	if err != nil {
		return nil, err
	}
	items, err := client.ListColumns(boardID, 1000)
	if err != nil {
		return nil, wrapCatalogErr(err)
	}
	out := make([]dto.YougileColumn, 0, len(items))
	for _, i := range items {
		if id, _ := i["id"].(string); id != "" {
			title, _ := i["title"].(string)
			col := dto.YougileColumn{ID: id, Title: title}
			if bid, _ := i["boardId"].(string); bid != "" {
				col.BoardID = &bid
			}
			out = append(out, col)
		}
	}
	return out, nil
}

// resolveFirstColumn — id первой колонки доски. «Первая» — та, что идёт
// первой в ответе /columns?boardId: явного поля order у YG нет, но на
// практике колонки возвращаются в порядке отображения; для нашего сценария
// (новые задачи в левой колонке) — годится.
func (y *Yougile) resolveFirstColumn(ctx context.Context, actorID int64, boardID string) (*string, error) {
	cols, err := y.Columns(ctx, actorID, boardID)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, nil
	}
	return &cols[0].ID, nil
}

// ── сохранение настроек ───────────────────────────────────────────────────

// UpdateCompanySettings — PUT /company-settings: частичное обновление. При
// смене доски автоматически перерезолвит yg_first_column_id; при очистке
// yg_company_id сбрасывает связанные поля (проект/доска/колонки).
func (y *Yougile) UpdateCompanySettings(ctx context.Context, actor *domain.User,
	companyID int64, upd dto.YougileSettingsUpdate) (*dto.YougileSettings, error) {

	company, err := y.repo.GetYougileCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.NewError("NO_COMPANY", "", 400)
	}

	changedBoard := upd.YgBoardIDSet &&
		strOrEmpty(normalizeOpt(upd.YgBoardID)) != strOrEmpty(company.YgBoardID)
	clearedCompany := upd.YgCompanyIDSet && normalizeOpt(upd.YgCompanyID) == nil

	fields := map[string]any{}
	setField := func(column string, dst **string, value *string) {
		v := normalizeOpt(value)
		*dst = v
		fields[column] = v
	}
	if upd.YgCompanyIDSet {
		setField("yg_company_id", &company.YgCompanyID, upd.YgCompanyID)
	}
	if upd.YgCompanyNameSet {
		setField("yg_company_name", &company.YgCompanyName, upd.YgCompanyName)
	}
	if upd.YgProjectIDSet {
		setField("yg_project_id", &company.YgProjectID, upd.YgProjectID)
	}
	if upd.YgProjectTitleSet {
		setField("yg_project_title", &company.YgProjectTitle, upd.YgProjectTitle)
	}
	if upd.YgBoardIDSet {
		setField("yg_board_id", &company.YgBoardID, upd.YgBoardID)
	}
	if upd.YgBoardTitleSet {
		setField("yg_board_title", &company.YgBoardTitle, upd.YgBoardTitle)
	}
	if upd.YgCompletedColumnIDSet {
		setField("yg_completed_column_id", &company.YgCompletedColumnID, upd.YgCompletedColumnID)
	}

	if clearedCompany {
		// Проект/доска/колонки без компании бессмысленны.
		for column, dst := range map[string]**string{
			"yg_project_id": &company.YgProjectID, "yg_project_title": &company.YgProjectTitle,
			"yg_board_id": &company.YgBoardID, "yg_board_title": &company.YgBoardTitle,
			"yg_first_column_id":     &company.YgFirstColumnID,
			"yg_completed_column_id": &company.YgCompletedColumnID,
		} {
			*dst = nil
			fields[column] = nil
		}
	}

	// Резолв первой колонки — при выборе доски, не дожидаясь отдельной
	// кнопки; board очищен — сбрасываем.
	if changedBoard {
		if company.YgBoardID != nil {
			first, err := y.resolveFirstColumn(ctx, actor.ID, *company.YgBoardID)
			if err != nil {
				y.log.Warn("yougile.resolve_first_col_failed",
					"company_id", companyID, "error", err)
				first = nil
			}
			company.YgFirstColumnID = first
			fields["yg_first_column_id"] = first
		} else {
			company.YgFirstColumnID = nil
			fields["yg_first_column_id"] = nil
		}
	}

	// Флаг включения в settings JSONB (UI: «фичу видно»).
	var enabledChangedTo *bool
	if upd.EnabledSet {
		if company.UsesYougile != upd.Enabled {
			v := upd.Enabled
			enabledChangedTo = &v
		}
		company.UsesYougile = upd.Enabled
	}

	if err := y.repo.UpdateYougileCompanyFields(ctx, companyID, fields); err != nil {
		return nil, err
	}
	if upd.EnabledSet {
		if err := y.repo.SetCompanyUsesYougile(ctx, companyID, upd.Enabled); err != nil {
			return nil, err
		}
	}

	// Webhook'и регистрируем/снимаем синхронно с переключением флага (ОК для
	// нашего объёма). Самолечение: интеграция включена, доска выбрана, а
	// yg_webhook_id ещё пуст (первое включение без PUBLIC_BASE) — пытаемся
	// зарегистрировать на каждое сохранение. Смена доски при живом webhook —
	// переподписка, иначе фильтр location остаётся на старой доске. Сетевые
	// ошибки YG логируем, но не блокируем сохранение настроек.
	needsRegister := company.UsesYougile && company.YgBoardID != nil && company.YgCompanyID != nil &&
		((enabledChangedTo != nil && *enabledChangedTo) ||
			company.YgWebhookID == nil || *company.YgWebhookID == "" ||
			changedBoard)
	if needsRegister {
		if err := y.ensureWebhookRegistered(ctx, actor.ID, company); err != nil {
			if werr := wrapMisconfig(err); werr == errEncKeyMisconfigured {
				return nil, werr
			}
			y.log.Warn("yougile.webhook_register_failed",
				"company_id", companyID, "error", err)
		}
	} else if enabledChangedTo != nil && !*enabledChangedTo {
		if err := y.deregisterWebhook(ctx, actor.ID, company); err != nil {
			if werr := wrapMisconfig(err); werr == errEncKeyMisconfigured {
				return nil, werr
			}
			y.log.Warn("yougile.webhook_deregister_failed_top",
				"company_id", companyID, "error", err)
		}
	}

	y.log.Info("yougile.company_settings_updated",
		"company_id", companyID, "actor_id", actor.ID)
	return settingsDTO(company), nil
}

// ResetIntegration — полный сброс «начать заново» (руководитель компании).
//
// Строгий порядок, пока ключ ещё на руках: 1) снимаем webhook в YouGile;
// 2) чистим конфигурацию компании и гасим uses_yougile; 3) отвязываем личный
// YouGile-аккаунт инициатора. Каждый внешний шаг best-effort: даже если
// YouGile недоступен, локальное состояние обнуляется — кнопка обязана
// сработать.
func (y *Yougile) ResetIntegration(ctx context.Context, actor *domain.User, companyID int64) (*dto.YougileSettings, error) {
	company, err := y.repo.GetYougileCompany(ctx, companyID)
	if err != nil {
		return nil, err
	}
	if company == nil {
		return nil, domain.NewError("NO_COMPANY", "", 400)
	}

	// 1. Webhook — до отвязки аккаунта, иначе нечем стучаться в YG.
	if err := y.deregisterWebhook(ctx, actor.ID, company); err != nil {
		y.log.Warn("yougile.reset_webhook_failed", "company_id", companyID, "error", err)
	}

	// 2. Конфигурация компании.
	fields := map[string]any{}
	for column, dst := range map[string]**string{
		"yg_company_id": &company.YgCompanyID, "yg_company_name": &company.YgCompanyName,
		"yg_project_id": &company.YgProjectID, "yg_project_title": &company.YgProjectTitle,
		"yg_board_id": &company.YgBoardID, "yg_board_title": &company.YgBoardTitle,
		"yg_first_column_id":     &company.YgFirstColumnID,
		"yg_completed_column_id": &company.YgCompletedColumnID,
		"yg_webhook_id":          &company.YgWebhookID,
		"yg_webhook_secret":      &company.YgWebhookSecret,
	} {
		*dst = nil
		fields[column] = nil
	}
	if err := y.repo.UpdateYougileCompanyFields(ctx, companyID, fields); err != nil {
		return nil, err
	}
	company.UsesYougile = false
	if err := y.repo.SetCompanyUsesYougile(ctx, companyID, false); err != nil {
		return nil, err
	}

	// 3. Личный аккаунт руководителя.
	if err := y.Disconnect(ctx, actor.ID); err != nil {
		y.log.Warn("yougile.reset_account_disconnect_failed",
			"company_id", companyID, "actor_id", actor.ID, "error", err)
	}

	y.log.Info("yougile.integration_reset", "company_id", companyID, "actor_id", actor.ID)
	return settingsDTO(company), nil
}
