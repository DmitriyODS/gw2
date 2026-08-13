package http

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/endpoint"
)

// bizZone — деловая таймзона платформы, та же, по которой SQL статистики режет
// сутки (`AT TIME ZONE 'Europe/Moscow'`). Фиксированное смещение, а не
// LoadLocation: tzdata в alpine-образе нет (как в assistant_stats.go и petsvc).
var bizZone = time.FixedZone("MSK", 3*60*60)

// parsePeriod — период отчёта; дефолт — текущий год целиком. Дата без времени —
// ДЕЛОВОЙ ДЕНЬ ЦЕЛИКОМ в МСК: дни в срезах бакетятся по МСК, и граница в UTC
// срезала бы у крайних дней первые три часа — «сегодня» теряло утренние
// поступления и закрытия. Значение со временем берётся как есть.
func parsePeriod(c *fiber.Ctx) (time.Time, time.Time, bool) {
	year := time.Now().In(bizZone).Year()
	start := time.Date(year, 1, 1, 0, 0, 0, 0, bizZone)
	end := time.Date(year, 12, 31, 23, 59, 59, 999999000, bizZone)

	fromStr, toStr := c.Query("from"), c.Query("to")
	if fromStr != "" {
		t, ok := parseISODateTime(fromStr)
		if !ok {
			return start, end, false
		}
		start = t
		if isDateOnly(fromStr) {
			start = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, bizZone)
		}
	}
	if toStr != "" {
		t, ok := parseISODateTime(toStr)
		if !ok {
			return start, end, false
		}
		end = t
		if isDateOnly(toStr) {
			end = time.Date(t.Year(), t.Month(), t.Day(),
				23, 59, 59, 999999000, bizZone)
		}
	}
	return start, end, true
}

func isDateOnly(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
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

// activityTarget — общий разбор преамбулы хендлеров активности: сотрудник из
// :id, период, актор.
func (h *handlers) activityTarget(c *fiber.Ctx) (endpoint.EmployeeActivityRequest, bool, error) {
	targetID, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return endpoint.EmployeeActivityRequest{}, false, scopeBadRequest(c, "Неверный id сотрудника")
	}
	start, end, ok := parsePeriod(c)
	if !ok {
		return endpoint.EmployeeActivityRequest{}, false, badPeriod(c)
	}
	return endpoint.EmployeeActivityRequest{
		Actor: currentUser(c), TargetUserID: targetID, Start: start, End: end,
	}, true, nil
}

func (h *handlers) employeeActivity(c *fiber.Ctx) error {
	req, ok, err := h.activityTarget(c)
	if !ok {
		return err
	}
	resp, err := h.eps.EmployeeActivity(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) employeeActivityFeed(c *fiber.Ctx) error {
	req, ok, err := h.activityTarget(c)
	if !ok {
		return err
	}
	req.Page, _ = strconv.Atoi(c.Query("page"))
	req.PerPage, _ = strconv.Atoi(c.Query("per_page"))
	resp, err := h.eps.EmployeeActivityFeed(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}

func (h *handlers) employeeActivityDocx(c *fiber.Ctx) error {
	req, ok, err := h.activityTarget(c)
	if !ok {
		return err
	}
	resp, err := h.eps.EmployeeActivityDocx(c.Context(), req)
	if err != nil {
		return h.respondError(c, err)
	}
	out := resp.(endpoint.EmployeeActivityDocxResponse)
	c.Set(fiber.HeaderContentType, "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	c.Set(fiber.HeaderContentDisposition,
		"attachment; filename=activity_"+req.Start.UTC().Format("2006-01-02")+"_"+req.End.UTC().Format("2006-01-02")+".docx")
	return c.Send(out.Data)
}

func (h *handlers) statsProfile(c *fiber.Ctx) error {
	start, end, ok := parsePeriod(c)
	if !ok {
		return badPeriod(c)
	}
	user := currentUser(c)
	resp, err := h.eps.StatsProfile(c.Context(), endpoint.ProfileRequest{
		UserID: user.ID, CompanyID: user.CompanyID, Start: start, End: end,
	})
	if err != nil {
		return h.respondError(c, err)
	}
	return c.JSON(resp)
}
