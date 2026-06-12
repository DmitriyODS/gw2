package http

import (
	"io"

	"github.com/gofiber/fiber/v2"
)

// Хендлеры резервной копии (порт back/app/api/backup.py): экспорт —
// ZIP-архив data.json + avatars/, импорт — ДЕСТРУКТИВНАЯ полная замена.

func (h *handlers) exportBackup(c *fiber.Ctx) error {
	resp, err := h.eps.ExportBackup(c.Context(), nil)
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

	// Любая ошибка восстановления — 400 IMPORT_ERROR, как try/except
	// вокруг import_zip во Flask.
	if _, err := h.eps.ImportBackup(c.Context(), zipBytes); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "IMPORT_ERROR", "message": "Ошибка импорта: " + err.Error(),
		})
	}
	return c.JSON(fiber.Map{"message": "Данные восстановлены"})
}
