package http

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/endpoint"
)

// queryInt — опциональный числовой query-параметр (как int(args[...]) во
// Flask, но мусор отвечает 400, а не 500).
func queryInt(c *fiber.Ctx, name string) (*int64, bool) {
	raw := c.Query(name)
	if raw == "" {
		return nil, true
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, false
	}
	return &v, true
}

func (h *handlers) listTasks(c *fiber.Ctx) error {
	user := currentUser(c)
	companyID, ok, err := requireCompanyScope(c, user)
	if !ok {
		return err
	}

	f := domain.TaskListFilter{
		CurrentUserID: user.ID,
		CompanyID:     &companyID,
		Tab:           c.Query("tab", "active"),
		Search:        c.Query("search"),
		Sort:          c.Query("sort", "last_activity"),
		HasUnits:      c.Query("has_units"),
		Page:          c.QueryInt("page", 1),
		PerPage:       c.QueryInt("per_page", 30),
	}
	for name, dst := range map[string]**int64{
		"dept_id": &f.DeptID, "stage_id": &f.StageID, "responsible_id": &f.ResponsibleUserID,
	} {
		v, ok := queryInt(c, name)
		if !ok {
			return scopeBadRequest(c, "Неверный "+name)
		}
		*dst = v
	}
	if raw := c.Query("received_from"); raw != "" {
		t, ok := parseISODateTime(raw)
		if !ok {
			return validationMsg(c, "Неверный формат даты")
		}
		f.ReceivedFrom = &t
	}
	if raw := c.Query("received_to"); raw != "" {
		t, ok := parseISODateTime(raw)
		if !ok {
			return validationMsg(c, "Неверный формат даты")
		}
		f.ReceivedTo = &t
	}
	if c.Query("created_by_me") == "1" {
		f.AuthorID = &user.ID
	}
	// tag_ids=1,2,3 — задачи, имеющие хотя бы один из выбранных тегов.
	if raw := c.Query("tag_ids"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
			if err != nil || id <= 0 {
				return scopeBadRequest(c, "Неверный tag_ids")
			}
			f.TagIDs = append(f.TagIDs, id)
		}
	}
	// colors=red,blue — задачи с моим личным цветом из выбранных.
	if raw := c.Query("colors"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			color := strings.TrimSpace(part)
			if !domain.ValidTaskColor(color) {
				return scopeBadRequest(c, "Неверный colors")
			}
			f.Colors = append(f.Colors, color)
		}
	}

	resp, err := h.eps.ListTasks(c.Context(), f)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) createTask(c *fiber.Ctx) error {
	user := currentUser(c)
	companyID, ok, err := requireCompanyScope(c, user)
	if !ok {
		return err
	}
	req, details := parseTaskCreate(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	resp, err := h.eps.CreateTask(c.Context(), endpoint.CreateTaskRequest{
		ActorID: user.ID, CompanyID: companyID, Body: req,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *handlers) getTask(c *fiber.Ctx) error {
	user := currentUser(c)
	resp, err := h.eps.GetTask(c.Context(), endpoint.TaskActorRequest{
		TaskID: pathID(c), ActorID: user.ID, CompanyID: user.CompanyID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) updateTask(c *fiber.Ctx) error {
	req, details := parseTaskUpdate(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	user := currentUser(c)
	resp, err := h.eps.UpdateTask(c.Context(), endpoint.UpdateTaskRequest{
		TaskID: pathID(c), ActorID: user.ID, CompanyID: user.CompanyID, Body: req,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteTask(c *fiber.Ctx) error {
	user := currentUser(c)
	if _, err := h.eps.DeleteTask(c.Context(), endpoint.TaskActorRequest{
		TaskID: pathID(c), ActorID: user.ID, CompanyID: user.CompanyID,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Задача удалена"})
}

func (h *handlers) archiveTask(c *fiber.Ctx) error {
	user := currentUser(c)
	resp, err := h.eps.ArchiveTask(c.Context(), endpoint.TaskActorRequest{
		TaskID: pathID(c), ActorID: user.ID, CompanyID: user.CompanyID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) restoreTask(c *fiber.Ctx) error {
	user := currentUser(c)
	resp, err := h.eps.RestoreTask(c.Context(), endpoint.TaskActorRequest{
		TaskID: pathID(c), ActorID: user.ID, CompanyID: user.CompanyID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) setTaskColor(c *fiber.Ctx) error {
	color, details := parseColorBody(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	taskID := pathID(c)
	user := currentUser(c)
	if _, err := h.eps.SetTaskColor(c.Context(), endpoint.TaskColorRequest{
		TaskID: taskID, UserID: user.ID, CompanyID: user.CompanyID, Color: color,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"task_id": taskID, "color": color})
}

func (h *handlers) toggleFavorite(c *fiber.Ctx) error {
	user := currentUser(c)
	isFav, err := h.eps.ToggleFavorite(c.Context(), endpoint.TaskActorRequest{
		TaskID: pathID(c), ActorID: user.ID, CompanyID: user.CompanyID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"is_favorite": isFav})
}

func (h *handlers) setResponsible(c *fiber.Ctx) error {
	value, details := parseNullableIntBody(c.Body(), "responsible_user_id")
	if details != nil {
		return validationError(c, details)
	}
	user := currentUser(c)
	resp, err := h.eps.SetResponsible(c.Context(), endpoint.SetResponsibleRequest{
		TaskID: pathID(c), ActorID: user.ID, CompanyID: user.CompanyID, ResponsibleUserID: value,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) setStage(c *fiber.Ctx) error {
	value, details := parseNullableIntBody(c.Body(), "stage_id")
	if details != nil {
		return validationError(c, details)
	}
	user := currentUser(c)
	resp, err := h.eps.SetStage(c.Context(), endpoint.SetStageRequest{
		TaskID: pathID(c), ActorID: user.ID, CompanyID: user.CompanyID, StageID: value,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) contributors(c *fiber.Ctx) error {
	user := currentUser(c)
	resp, err := h.eps.Contributors(c.Context(), endpoint.TaskActorRequest{
		TaskID: pathID(c), ActorID: user.ID, CompanyID: user.CompanyID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"items": resp})
}

// ── Комментарии задач ────────────────────────────────────────────

func (h *handlers) listComments(c *fiber.Ctx) error {
	user := currentUser(c)
	resp, err := h.eps.ListComments(c.Context(), endpoint.TaskActorRequest{
		TaskID: pathID(c), ActorID: user.ID, CompanyID: user.CompanyID,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) seenComments(c *fiber.Ctx) error {
	user := currentUser(c)
	if _, err := h.eps.MarkCommentsSeen(c.Context(), endpoint.TaskActorRequest{
		TaskID: pathID(c), ActorID: user.ID, CompanyID: user.CompanyID,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *handlers) createComment(c *fiber.Ctx) error {
	text, details := parseTextBody(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	user := currentUser(c)
	resp, err := h.eps.CreateComment(c.Context(), endpoint.CommentCreateRequest{
		TaskID: pathID(c), AuthorID: user.ID, CompanyID: user.CompanyID, Text: text,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func commentID(c *fiber.Ctx) int64 {
	id, _ := c.ParamsInt("commentId")
	return int64(id)
}

func (h *handlers) updateComment(c *fiber.Ctx) error {
	text, details := parseTextBody(c.Body())
	if details != nil {
		return validationError(c, details)
	}
	user := currentUser(c)
	resp, err := h.eps.UpdateComment(c.Context(), endpoint.CommentEditRequest{
		TaskID: pathID(c), CommentID: commentID(c), UserID: user.ID,
		ActorLevel: user.RoleLevel, CompanyID: user.CompanyID, Text: text,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) deleteComment(c *fiber.Ctx) error {
	user := currentUser(c)
	if _, err := h.eps.DeleteComment(c.Context(), endpoint.CommentEditRequest{
		TaskID: pathID(c), CommentID: commentID(c), UserID: user.ID,
		ActorLevel: user.RoleLevel, CompanyID: user.CompanyID,
	}); err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(fiber.Map{"message": "Удалён"})
}
