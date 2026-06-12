package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/endpoint"
)

func (h *handlers) activeUnit(c *fiber.Ctx) error {
	resp, err := h.eps.ActiveUnit(c.Context(), currentUser(c).ID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) taskUnits(c *fiber.Ctx) error {
	resp, err := h.eps.TaskUnits(c.Context(), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createUnit(c *fiber.Ctx) error {
	name, unitTypeID, details := parseUnitCreate(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.CreateUnit(c.Context(), endpoint.CreateUnitRequest{
		TaskID: pathID(c), UserID: currentUser(c).ID, Name: name, UnitTypeID: unitTypeID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) updateUnit(c *fiber.Ctx) error {
	req, details := parseUnitUpdate(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	user := currentUser(c)
	resp, err := h.eps.UpdateUnit(c.Context(), endpoint.UpdateUnitRequest{
		UnitID: pathID(c), ActorID: user.ID, ActorLevel: user.RoleLevel, Body: req,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) stopUnit(c *fiber.Ctx) error {
	user := currentUser(c)
	resp, err := h.eps.StopUnit(c.Context(), endpoint.UnitActorRequest{
		UnitID: pathID(c), ActorID: user.ID, ActorLevel: user.RoleLevel,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteUnit(c *fiber.Ctx) error {
	user := currentUser(c)
	if _, err := h.eps.DeleteUnit(c.Context(), endpoint.UnitActorRequest{
		UnitID: pathID(c), ActorID: user.ID, ActorLevel: user.RoleLevel,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Юнит удалён"})
}
