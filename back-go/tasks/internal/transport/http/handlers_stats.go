package http

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/endpoint"
)

// parsePeriod — как _parse_period во Flask: дефолт — текущий год целиком
// (UTC); date-only `to` расширяется до конца дня.
func parsePeriod(c *fiber.Ctx) (time.Time, time.Time, bool) {
	year := time.Now().UTC().Year()
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)

	fromStr, toStr := c.Query("from"), c.Query("to")
	if fromStr != "" {
		t, ok := parseISODateTime(fromStr)
		if !ok {
			return start, end, false
		}
		start = t
	}
	if toStr != "" {
		t, ok := parseISODateTime(toStr)
		if !ok {
			return start, end, false
		}
		end = t
		if !containsT(toStr) {
			end = time.Date(end.Year(), end.Month(), end.Day(),
				23, 59, 59, 999999000, end.Location())
		}
	}
	return start, end, true
}

func containsT(s string) bool {
	for _, r := range s {
		if r == 'T' {
			return true
		}
	}
	return false
}

func badPeriod(c *fiber.Ctx) error {
	return validationMsg(c, "Неверный формат даты. Используйте YYYY-MM-DD")
}

// statsPeriodRequest — общая преамбула stats-хендлеров: период + scope.
func (h *handlers) statsPeriodRequest(c *fiber.Ctx) (endpoint.PeriodRequest, bool, error) {
	start, end, ok := parsePeriod(c)
	if !ok {
		return endpoint.PeriodRequest{}, false, badPeriod(c)
	}
	companyID, ok, err := optionalCompanyScope(c, currentUser(c))
	if !ok {
		return endpoint.PeriodRequest{}, false, err
	}
	return endpoint.PeriodRequest{Start: start, End: end, CompanyID: companyID}, true, nil
}

func (h *handlers) statsCommon(c *fiber.Ctx) error {
	req, ok, err := h.statsPeriodRequest(c)
	if !ok {
		return err
	}
	resp, err := h.eps.StatsCommon(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) statsExtended(c *fiber.Ctx) error {
	req, ok, err := h.statsPeriodRequest(c)
	if !ok {
		return err
	}
	resp, err := h.eps.StatsExtended(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

const xlsxMime = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

func (h *handlers) sendXLSX(c *fiber.Ctx, name string, req endpoint.PeriodRequest, data []byte) error {
	c.Set(fiber.HeaderContentType, xlsxMime)
	c.Set(fiber.HeaderContentDisposition,
		"attachment; filename=stats_"+name+"_"+req.Start.UTC().Format("2006-01-02")+
			"_"+req.End.UTC().Format("2006-01-02")+".xlsx")
	return c.Send(data)
}

func (h *handlers) exportCommon(c *fiber.Ctx) error {
	req, ok, err := h.statsPeriodRequest(c)
	if !ok {
		return err
	}
	resp, err := h.eps.ExportCommonXLSX(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	return h.sendXLSX(c, "common", req, resp.([]byte))
}

func (h *handlers) exportExtended(c *fiber.Ctx) error {
	req, ok, err := h.statsPeriodRequest(c)
	if !ok {
		return err
	}
	resp, err := h.eps.ExportExtendedXLSX(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	return h.sendXLSX(c, "extended", req, resp.([]byte))
}

func (h *handlers) statsUserTasks(c *fiber.Ctx) error {
	start, end, ok := parsePeriod(c)
	if !ok {
		return badPeriod(c)
	}
	user := currentUser(c)
	targetUserID := user.ID
	if raw := c.Query("user_id"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return scopeBadRequest(c, "Неверный user_id")
		}
		targetUserID = v
	}
	resp, err := h.eps.StatsUserTasks(c.Context(), endpoint.UserTasksRequest{
		Actor: user, TargetUserID: targetUserID, Start: start, End: end,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) statsEmployees(c *fiber.Ctx) error {
	companyID, ok, err := optionalCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	resp, err := h.eps.StatsEmployees(c.Context(), companyID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) statsResponsibles(c *fiber.Ctx) error {
	companyID, ok, err := optionalCompanyScope(c, currentUser(c))
	if !ok {
		return err
	}
	resp, err := h.eps.StatsResponsibles(c.Context(), companyID)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) statsProfile(c *fiber.Ctx) error {
	start, end, ok := parsePeriod(c)
	if !ok {
		return badPeriod(c)
	}
	resp, err := h.eps.StatsProfile(c.Context(), endpoint.ProfileRequest{
		UserID: currentUser(c).ID, Start: start, End: end,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}
