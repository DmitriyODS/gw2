package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/endpoint"
)

// Справочники компании: типы юнитов, отделы, этапы. Все роуты в
// company-scope (как @require_company_scope во Flask).

func (h *handlers) listUnitTypes(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	resp, err := h.eps.ListUnitTypes(c.Context(), companyID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createUnitType(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	name, details := parseNameBody(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.CreateUnitType(c.Context(), endpoint.CompanyNameRequest{
		CompanyID: companyID, Name: name,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateUnitType(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	name, details := parseNameBody(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.UpdateUnitType(c.Context(), endpoint.CompanyNameRequest{
		CompanyID: companyID, ItemID: pathID(c), Name: name,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteUnitType(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	if _, err := h.eps.DeleteUnitType(c.Context(), endpoint.CompanyItemRequest{
		CompanyID: companyID, ItemID: pathID(c),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Тип юнита удалён"})
}

func (h *handlers) listDepartments(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	resp, err := h.eps.ListDepartments(c.Context(), companyID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createDepartment(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	name, details := parseNameBody(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.CreateDepartment(c.Context(), endpoint.CompanyNameRequest{
		CompanyID: companyID, Name: name,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateDepartment(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	name, details := parseNameBody(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.UpdateDepartment(c.Context(), endpoint.CompanyNameRequest{
		CompanyID: companyID, ItemID: pathID(c), Name: name,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteDepartment(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	if _, err := h.eps.DeleteDepartment(c.Context(), endpoint.CompanyItemRequest{
		CompanyID: companyID, ItemID: pathID(c),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Отдел удалён"})
}

func (h *handlers) listStages(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	resp, err := h.eps.ListStages(c.Context(), companyID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createStage(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	name, color, details := parseStageCreate(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.CreateStage(c.Context(), endpoint.StageCreateRequest{
		CompanyID: companyID, Name: name, Color: color,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateStage(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	name, color, details := parseStageUpdate(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.UpdateStage(c.Context(), endpoint.StageUpdateRequest{
		CompanyID: companyID, StageID: pathID(c), Name: name, Color: color,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteStage(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	if _, err := h.eps.DeleteStage(c.Context(), endpoint.CompanyItemRequest{
		CompanyID: companyID, ItemID: pathID(c),
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Этап удалён"})
}

func (h *handlers) reorderStages(c *fiber.Ctx) error {
	companyID, ok, err := requireCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	ids, details := parseReorder(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.ReorderStages(c.Context(), endpoint.ReorderRequest{
		CompanyID: companyID, IDs: ids,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}
