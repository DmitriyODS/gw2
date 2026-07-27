package http

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/reminder/internal/domain"
)

// reminderBody — тело создания/правки. Указатели различают «поле не пришло» и
// «поле очищено» (частичный PATCH).
type reminderBody struct {
	Title    *string        `json:"title"`
	Note     *string        `json:"note"`
	RemindAt *time.Time     `json:"remind_at"`
	Timezone *string        `json:"timezone"`
	Repeat   *domain.Repeat `json:"repeat"`
	Link     *domain.Link   `json:"link"`
	Active   *bool          `json:"active"`
}

func parseBody(c *fiber.Ctx, dst any) error {
	if err := c.BodyParser(dst); err != nil {
		return domain.NewError("VALIDATION", "Некорректное тело запроса", 400)
	}
	return nil
}

func (h *handlers) list(c *fiber.Ctx) error {
	items, err := h.svc.List(c.Context(), currentUserID(c), domain.ListScope(c.Query("scope")))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *handlers) upcoming(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit"))
	items, err := h.svc.Upcoming(c.Context(), currentUserID(c), limit)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *handlers) linked(c *fiber.Ctx) error {
	recordID, _ := strconv.ParseInt(c.Query("record_id"), 10, 64)
	items, err := h.svc.ByLink(c.Context(), currentUserID(c), c.Query("kind"), recordID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *handlers) get(c *fiber.Ctx) error {
	r, err := h.svc.Get(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(r)
}

func (h *handlers) create(c *fiber.Ctx) error {
	var body reminderBody
	if err := parseBody(c, &body); err != nil {
		return h.respondError(c, err)
	}
	r := &domain.Reminder{OwnerID: currentUserID(c)}
	if body.Title != nil {
		r.Title = *body.Title
	}
	if body.Note != nil {
		r.Note = *body.Note
	}
	if body.RemindAt != nil {
		r.RemindAt = *body.RemindAt
	}
	if body.Timezone != nil {
		r.Timezone = *body.Timezone
	}
	if body.Repeat != nil {
		r.Repeat = *body.Repeat
	}
	if body.Link != nil {
		r.Link = *body.Link
	}
	created, err := h.svc.Create(c.Context(), r)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(created)
}

func (h *handlers) update(c *fiber.Ctx) error {
	var body reminderBody
	if err := parseBody(c, &body); err != nil {
		return h.respondError(c, err)
	}
	r, err := h.svc.Update(c.Context(), currentUserID(c), pathID(c), domain.ReminderUpdate{
		Title: body.Title, Note: body.Note, RemindAt: body.RemindAt,
		Timezone: body.Timezone, Repeat: body.Repeat, Link: body.Link, Active: body.Active,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(r)
}

func (h *handlers) remove(c *fiber.Ctx) error {
	if err := h.svc.Delete(c.Context(), currentUserID(c), pathID(c)); err != nil {
		return h.respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *handlers) snooze(c *fiber.Ctx) error {
	var body struct {
		Minutes int `json:"minutes"`
	}
	if err := parseBody(c, &body); err != nil {
		return h.respondError(c, err)
	}
	if body.Minutes == 0 {
		body.Minutes = 10
	}
	r, err := h.svc.Snooze(c.Context(), currentUserID(c), pathID(c), body.Minutes)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(r)
}

func (h *handlers) done(c *fiber.Ctx) error {
	r, err := h.svc.Complete(c.Context(), currentUserID(c), pathID(c))
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(r)
}
