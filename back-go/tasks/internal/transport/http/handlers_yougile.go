package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/endpoint"
)

// trimQuery — query-параметр со strip(), как (args.get(...) or "").strip().
func trimQuery(c *fiber.Ctx, name string) string {
	return strings.TrimSpace(c.Query(name))
}

// Хендлеры /api/yougile/* — пути и формы JSON байт-в-байт с прежним
// Flask-блюпринтом api/yougile.py.

func (h *handlers) yougileStatus(c *fiber.Ctx) error {
	resp, err := h.eps.YougileStatus(c.Context(), currentUser(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// yougileConnect — подключение аккаунта. Обычный юзер не может выбрать
// произвольную yg_company (она зафиксирована настройками компании) — поле
// yg_company_id учитывается только для DIRECTOR+ (админ в визарде выбирает
// будущую компанию до сохранения настроек), иначе молча игнорируется.
func (h *handlers) yougileConnect(c *fiber.Ctx) error {
	req, details := parseYougileConnect(c.Body())
	if details != nil {
		return yougileValidationError(c, details)
	}
	user := currentUser(c)
	var explicit *string
	if user.RoleLevel >= domain.LevelDirector {
		explicit = req.YgCompanyID
	}
	resp, err := h.eps.YougileConnect(c.Context(), endpoint.YougileConnectRequest{
		User: user, Login: req.Login, Password: req.Password, Explicit: explicit,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) yougileDisconnect(c *fiber.Ctx) error {
	if _, err := h.eps.YougileDisconnect(c.Context(), currentUser(c).ID); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"connected": false})
}

func (h *handlers) yougileRotate(c *fiber.Ctx) error {
	password, details := parseYougileRotate(c.Body())
	if details != nil {
		return yougileValidationError(c, details)
	}
	resp, err := h.eps.YougileRotate(c.Context(), endpoint.YougileRotateRequest{
		User: currentUser(c), Password: password,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// yougileLookupCompanies — `POST /auth/companies` под капотом: отдаём
// админу выбор YG-компании.
func (h *handlers) yougileLookupCompanies(c *fiber.Ctx) error {
	login, password, details := parseYougileConnectStart(c.Body())
	if details != nil {
		return yougileValidationError(c, details)
	}
	resp, err := h.eps.YougileLookupCompanies(c.Context(), endpoint.YougileCredsRequest{
		Login: login, Password: password,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) yougileProjects(c *fiber.Ctx) error {
	resp, err := h.eps.YougileProjects(c.Context(), currentUser(c).ID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) yougileBoards(c *fiber.Ctx) error {
	projectID := trimQuery(c, "projectId")
	if projectID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "VALIDATION", "message": "Нужен параметр projectId",
		})
	}
	resp, err := h.eps.YougileBoards(c.Context(), endpoint.YougileCatalogRequest{
		ActorID: currentUser(c).ID, Param: projectID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) yougileColumns(c *fiber.Ctx) error {
	boardID := trimQuery(c, "boardId")
	if boardID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "VALIDATION", "message": "Нужен параметр boardId",
		})
	}
	resp, err := h.eps.YougileColumns(c.Context(), endpoint.YougileCatalogRequest{
		ActorID: currentUser(c).ID, Param: boardID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// yougileGetSettings — без компании (root admin до выбора) отдаём пустые
// настройки, чтобы фронт мог отрендерить визард, а не ловить 400.
func (h *handlers) yougileGetSettings(c *fiber.Ctx) error {
	resp, err := h.eps.YougileGetSettings(c.Context(), currentUser(c).CompanyID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// requireOwnCompany — компания пользователя обязательна (как
// _own_company_or_403 во Flask); nil — ответ уже записан.
func requireOwnCompany(c *fiber.Ctx, user *domain.User) (*int64, error) {
	if user.CompanyID == nil {
		return nil, c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "NO_COMPANY"})
	}
	return user.CompanyID, nil
}

func (h *handlers) yougilePutSettings(c *fiber.Ctx) error {
	user := currentUser(c)
	companyID, err := requireOwnCompany(c, user)
	if companyID == nil {
		return err
	}
	req, details := parseYougileSettingsUpdate(c.Body())
	if details != nil {
		return yougileValidationError(c, details)
	}
	resp, err := h.eps.YougileUpdateSettings(c.Context(), endpoint.YougileSettingsUpdateRequest{
		Actor: user, CompanyID: *companyID, Body: req,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// yougileReset — полный сброс интеграции «начать заново».
func (h *handlers) yougileReset(c *fiber.Ctx) error {
	user := currentUser(c)
	companyID, err := requireOwnCompany(c, user)
	if companyID == nil {
		return err
	}
	resp, err := h.eps.YougileReset(c.Context(), endpoint.YougileCompanyActorRequest{
		Actor: user, CompanyID: *companyID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) yougileImportTask(c *fiber.Ctx) error {
	req, details := parseYougileImport(c.Body())
	if details != nil {
		return yougileValidationError(c, details)
	}
	resp, err := h.eps.YougileImport(c.Context(), endpoint.YougileImportRequest{
		User: currentUser(c), Body: req, Origin: c.BaseURL(),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) yougileExportTask(c *fiber.Ctx) error {
	taskID, details := parseYougileExport(c.Body())
	if details != nil {
		return yougileValidationError(c, details)
	}
	resp, err := h.eps.YougileExport(c.Context(), endpoint.YougileTaskActionRequest{
		User: currentUser(c), TaskID: taskID, Origin: c.BaseURL(),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// yougileUnlinkTask — разорвать связь GW-задачи с YouGile (карточка в YG
// не удаляется).
func (h *handlers) yougileUnlinkTask(c *fiber.Ctx) error {
	resp, err := h.eps.YougileUnlink(c.Context(), endpoint.YougileTaskActionRequest{
		User: currentUser(c), TaskID: pathID(c),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

// yougileWebhook — приём событий YouGile: без токена, авторизация через
// secret в URL. Неверная пара company/secret — просто 404 для
// злоумышленника; сбойные события не роняют batch (200 + results).
func (h *handlers) yougileWebhook(c *fiber.Ctx) error {
	companyID, _ := c.ParamsInt("companyId")
	resp, err := h.eps.YougileWebhook(c.Context(), endpoint.YougileWebhookRequest{
		CompanyID: int64(companyID), Secret: c.Params("secret"),
		Body: append([]byte(nil), c.Body()...),
	})
	if err != nil {
		return h.respondError(c, err)
	}
	res := resp.(endpoint.YougileWebhookResponse)
	if !res.Found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "NOT_FOUND"})
	}
	return c.JSON(fiber.Map{"results": res.Results})
}

// yougileRegisterWebhook — ручная регистрация webhook'а («сбросилось»/
// «поменяли URL»).
func (h *handlers) yougileRegisterWebhook(c *fiber.Ctx) error {
	user := currentUser(c)
	companyID, err := requireOwnCompany(c, user)
	if companyID == nil {
		return err
	}
	resp, err := h.eps.YougileRegisterWebhook(c.Context(), endpoint.YougileCompanyActorRequest{
		Actor: user, CompanyID: *companyID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"webhook_registered": resp})
}
