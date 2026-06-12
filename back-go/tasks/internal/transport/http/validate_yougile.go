package http

import (
	"encoding/json"
	"regexp"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/pkg/marshform"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

// Формы /api/yougile (порт schemas/yougile.py). Ошибки валидации здесь —
// {"error": "VALIDATION", "details": {...}} (так отвечали роуты Flask,
// в отличие от VALIDATION_ERROR/message у задач).

func yougileValidationError(c *fiber.Ctx, details map[string]any) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "VALIDATION", "details": details,
	})
}

// emailRE — практичная проверка fields.Email (полный RFC-регэксп marshmallow
// не воспроизводим; сообщение — его же).
var emailRE = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

const msgNotEmail = "Not a valid email address."

func validateLogin(value json.RawMessage, details map[string]any) string {
	s, ok := marshform.AsString(value)
	switch {
	case !ok:
		details["login"] = []string{marshform.MsgNotString}
	case !emailRE.MatchString(s):
		details["login"] = []string{msgNotEmail}
	case utf8.RuneCountInString(s) > 255:
		details["login"] = []string{marshform.LengthMax(255)}
	default:
		return s
	}
	return ""
}

func validatePassword(value json.RawMessage, details map[string]any) string {
	s, ok := marshform.AsString(value)
	switch {
	case !ok:
		details["password"] = []string{marshform.MsgNotString}
	case utf8.RuneCountInString(s) < 1:
		details["password"] = []string{marshform.LengthMin(1)}
	case utf8.RuneCountInString(s) > 512:
		details["password"] = []string{marshform.LengthMax(512)}
	default:
		return s
	}
	return ""
}

// parseYougileConnectStart — YougileConnectStartSchema: login + password.
func parseYougileConnectStart(body []byte) (login, password string, details map[string]any) {
	raw := rawBody(body)
	details = map[string]any{}
	if value, ok := raw["login"]; ok {
		login = validateLogin(value, details)
	} else {
		details["login"] = []string{marshform.MsgRequired}
	}
	if value, ok := raw["password"]; ok {
		password = validatePassword(value, details)
	} else {
		details["password"] = []string{marshform.MsgRequired}
	}
	for field := range raw {
		if field != "login" && field != "password" {
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return "", "", details
	}
	return login, password, nil
}

// parseYougileConnect — YougileConnectFinishSchema: + yg_company_id
// (load_default None).
func parseYougileConnect(body []byte) (dto.YougileConnect, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	var req dto.YougileConnect
	if value, ok := raw["login"]; ok {
		req.Login = validateLogin(value, details)
	} else {
		details["login"] = []string{marshform.MsgRequired}
	}
	if value, ok := raw["password"]; ok {
		req.Password = validatePassword(value, details)
	} else {
		details["password"] = []string{marshform.MsgRequired}
	}
	for field, value := range raw {
		switch field {
		case "login", "password":
		case "yg_company_id":
			if marshform.IsNull(value) {
				continue
			}
			s, ok := marshform.AsString(value)
			switch {
			case !ok:
				details[field] = []string{marshform.MsgNotString}
			case utf8.RuneCountInString(s) > 64:
				details[field] = []string{marshform.LengthMax(64)}
			default:
				req.YgCompanyID = &s
			}
		default:
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return dto.YougileConnect{}, details
	}
	return req, nil
}

// parseYougileRotate — YougileRotateSchema: password.
func parseYougileRotate(body []byte) (string, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	password := ""
	if value, ok := raw["password"]; ok {
		password = validatePassword(value, details)
	} else {
		details["password"] = []string{marshform.MsgRequired}
	}
	for field := range raw {
		if field != "password" {
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return "", details
	}
	return password, nil
}

// parseYougileSettingsUpdate — YougileCompanySettingsUpdateSchema: все поля
// опциональны (частичное обновление), строки allow_none.
func parseYougileSettingsUpdate(body []byte) (dto.YougileSettingsUpdate, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	var req dto.YougileSettingsUpdate

	parseStr := func(field string, maxLen int, dst **string, set *bool) {
		value := raw[field]
		*set = true
		if marshform.IsNull(value) {
			return
		}
		s, ok := marshform.AsString(value)
		switch {
		case !ok:
			*set = false
			details[field] = []string{marshform.MsgNotString}
		case utf8.RuneCountInString(s) > maxLen:
			*set = false
			details[field] = []string{marshform.LengthMax(maxLen)}
		default:
			*dst = &s
		}
	}

	for field, value := range raw {
		switch field {
		case "enabled":
			if v, ok := marshform.AsBool(value); ok {
				req.Enabled, req.EnabledSet = v, true
			} else {
				details[field] = []string{marshform.MsgNotBoolean}
			}
		case "yg_company_id":
			parseStr(field, 64, &req.YgCompanyID, &req.YgCompanyIDSet)
		case "yg_company_name":
			parseStr(field, 255, &req.YgCompanyName, &req.YgCompanyNameSet)
		case "yg_project_id":
			parseStr(field, 64, &req.YgProjectID, &req.YgProjectIDSet)
		case "yg_project_title":
			parseStr(field, 255, &req.YgProjectTitle, &req.YgProjectTitleSet)
		case "yg_board_id":
			parseStr(field, 64, &req.YgBoardID, &req.YgBoardIDSet)
		case "yg_board_title":
			parseStr(field, 255, &req.YgBoardTitle, &req.YgBoardTitleSet)
		case "yg_completed_column_id":
			parseStr(field, 64, &req.YgCompletedColumnID, &req.YgCompletedColumnIDSet)
		default:
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return dto.YougileSettingsUpdate{}, details
	}
	return req, nil
}

// parseYougileImport — YougileImportTaskSchema.
func parseYougileImport(body []byte) (dto.YougileImport, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	req := dto.YougileImport{PullDeadline: true}

	if value, ok := raw["url"]; ok {
		s, sok := marshform.AsString(value)
		switch {
		case !sok:
			details["url"] = []string{marshform.MsgNotString}
		case utf8.RuneCountInString(s) < 1:
			details["url"] = []string{marshform.LengthMin(1)}
		case utf8.RuneCountInString(s) > 2000:
			details["url"] = []string{marshform.LengthMax(2000)}
		default:
			req.URL = s
		}
	} else {
		details["url"] = []string{marshform.MsgRequired}
	}
	if value, ok := raw["department_id"]; ok {
		if v, vok := marshform.AsInt(value); vok {
			req.DepartmentID = v
		} else {
			details["department_id"] = []string{marshform.MsgNotInteger}
		}
	} else {
		details["department_id"] = []string{marshform.MsgRequired}
	}

	for field, value := range raw {
		switch field {
		case "url", "department_id":
		case "responsible_user_id", "stage_id":
			if marshform.IsNull(value) {
				continue
			}
			v, ok := marshform.AsInt(value)
			if !ok {
				details[field] = []string{marshform.MsgNotInteger}
				continue
			}
			if field == "responsible_user_id" {
				req.ResponsibleUserID = &v
			} else {
				req.StageID = &v
			}
		case "pull_deadline":
			if v, ok := marshform.AsBool(value); ok {
				req.PullDeadline = v
			} else {
				details[field] = []string{marshform.MsgNotBoolean}
			}
		default:
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return dto.YougileImport{}, details
	}
	return req, nil
}

// parseYougileExport — YougileExportTaskSchema: gw_task_id required.
func parseYougileExport(body []byte) (int64, map[string]any) {
	raw := rawBody(body)
	details := map[string]any{}
	var taskID int64
	if value, ok := raw["gw_task_id"]; ok {
		if v, vok := marshform.AsInt(value); vok {
			taskID = v
		} else {
			details["gw_task_id"] = []string{marshform.MsgNotInteger}
		}
	} else {
		details["gw_task_id"] = []string{marshform.MsgRequired}
	}
	for field := range raw {
		if field != "gw_task_id" {
			details[field] = []string{marshform.MsgUnknownField}
		}
	}
	if len(details) > 0 {
		return 0, details
	}
	return taskID, nil
}
