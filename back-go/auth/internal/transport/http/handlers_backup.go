package http

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/endpoint"
)

// Хендлеры резервной копии (только супер-админ): экспорт — ZIP data.json +
// avatars/ по выбранным разделам, импорт — ДЕСТРУКТИВНАЯ замена выбранных
// разделов. Разделы передаёт фронт (модалки выбора); пусто — все разделы.

func (h *handlers) exportBackup(c *fiber.Ctx) error {
	sections := parseSectionsQuery(c)
	resp, err := h.eps.ExportBackup(c.Context(), sections)
	if err != nil {
		return h.respondError(c, err)
	}
	c.Set(fiber.HeaderContentType, "application/zip")
	c.Set(fiber.HeaderContentDisposition, "attachment; filename=grovework_backup.zip")
	return c.Send(resp.([]byte))
}

func (h *handlers) importBackup(c *fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "NO_FILE", "message": "Файл не передан",
		})
	}
	f, err := fileHeader.Open()
	if err != nil {
		return h.respondError(c, err)
	}
	defer f.Close() //nolint:errcheck
	zipBytes, err := io.ReadAll(f)
	if err != nil {
		return h.respondError(c, err)
	}

	sections := parseSectionsForm(c)

	if _, err := h.eps.ImportBackup(c.Context(), endpoint.ImportBackupReq{Zip: zipBytes, Sections: sections}); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "IMPORT_ERROR", "message": "Ошибка импорта: " + err.Error(),
		})
	}
	return c.JSON(fiber.Map{"message": "Данные восстановлены"})
}

// parseSectionsQuery — секции из query (?sections=a,b,c). Пусто — nil (все).
func parseSectionsQuery(c *fiber.Ctx) []string {
	return splitSections(c.Query("sections"))
}

// parseSectionsForm — секции из multipart-поля sections (CSV или JSON-массив).
func parseSectionsForm(c *fiber.Ctx) []string {
	v := c.FormValue("sections")
	if strings.HasPrefix(strings.TrimSpace(v), "[") {
		var arr []string
		if json.Unmarshal([]byte(v), &arr) == nil {
			return arr
		}
	}
	return splitSections(v)
}

func splitSections(v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
